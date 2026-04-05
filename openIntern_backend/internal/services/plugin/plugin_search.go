package plugin

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"openIntern/internal/dao"
	"openIntern/internal/util"
)

const (
	defaultToolSearchTopK        = 10
	maxToolSearchTopK            = 10
	defaultToolSearchMaxMCPTools = 3
)

var camelCaseBoundaryPattern = regexp.MustCompile(`([a-z0-9])([A-Z])`)

// ToolSearchOptions 表示 search 模式下工具召回参数。
type ToolSearchOptions struct {
	TopK         int
	RuntimeTypes []string
	MaxMCPTools  int
}

// RuntimeToolSearchMatch 表示本地检索命中的最小工具元信息。
type RuntimeToolSearchMatch struct {
	ToolID   string
	ToolName string
}

// localRuntimeToolCandidate 表示本地关键词检索所需的工具候选项。
type localRuntimeToolCandidate struct {
	ToolID            string
	ToolName          string
	Description       string
	RuntimeType       string
	PluginName        string
	PluginDescription string
	NameParts         []string
	FullName          string
}

// parsedToolSearchQuery 保存 Claude Code 风格查询的解析结果。
type parsedToolSearchQuery struct {
	Raw           string
	IsSelect      bool
	SelectedNames []string
	RequiredTerms []string
	OptionalTerms []string
}

type scoredToolCandidate struct {
	Candidate localRuntimeToolCandidate
	Score     int
}

// SearchRuntimeToolIDs 通过本地关键词匹配返回保序的 tool_id 结果。
func (s *PluginService) SearchRuntimeToolIDs(ctx context.Context, query string, options ToolSearchOptions) ([]string, error) {
	matches, err := s.SearchRuntimeTools(ctx, query, options)
	if err != nil {
		return nil, err
	}

	toolIDs := make([]string, 0, len(matches))
	for _, item := range matches {
		toolID := strings.TrimSpace(item.ToolID)
		if toolID == "" {
			continue
		}
		toolIDs = append(toolIDs, toolID)
	}
	return util.NormalizeUniqueStringList(toolIDs), nil
}

// SearchRuntimeTools 使用本地候选工具快照执行 Claude Code 风格检索。
func (s *PluginService) SearchRuntimeTools(ctx context.Context, query string, options ToolSearchOptions) ([]RuntimeToolSearchMatch, error) {
	userID := userIDFromContext(ctx)
	if userID == "" {
		return []RuntimeToolSearchMatch{}, nil
	}

	parsed, ok := parseToolSearchQuery(query)
	if !ok {
		return []RuntimeToolSearchMatch{}, nil
	}

	runtimeTypes := normalizeToolSearchRuntimeTypes(options.RuntimeTypes)
	rows, err := dao.Plugin.ListEnabledRuntimeToolSearchRows(userID, runtimeTypes)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []RuntimeToolSearchMatch{}, nil
	}

	candidates := buildLocalRuntimeToolCandidates(rows)
	if len(candidates) == 0 {
		return []RuntimeToolSearchMatch{}, nil
	}

	topK := normalizeToolSearchTopK(options.TopK)
	maxMCPTools := normalizeToolSearchMaxMCPTools(options.MaxMCPTools)

	if parsed.IsSelect {
		return selectRuntimeToolsByName(candidates, parsed.SelectedNames, topK, maxMCPTools), nil
	}

	return searchRuntimeToolsByKeywords(candidates, parsed, topK, maxMCPTools), nil
}

// parseToolSearchQuery 解析 select 查询与 +required 语法。
func parseToolSearchQuery(query string) (parsedToolSearchQuery, bool) {
	raw := strings.TrimSpace(query)
	if raw == "" {
		return parsedToolSearchQuery{}, false
	}

	lowered := strings.ToLower(raw)
	if strings.HasPrefix(lowered, "select:") {
		selectedNames := make([]string, 0)
		for _, item := range strings.Split(raw[len("select:"):], ",") {
			name := strings.TrimSpace(item)
			if name == "" {
				continue
			}
			selectedNames = append(selectedNames, name)
		}
		selectedNames = util.NormalizeUniqueStringList(selectedNames)
		if len(selectedNames) == 0 {
			return parsedToolSearchQuery{}, false
		}
		return parsedToolSearchQuery{
			Raw:           raw,
			IsSelect:      true,
			SelectedNames: selectedNames,
		}, true
	}

	requiredTerms := make([]string, 0)
	optionalTerms := make([]string, 0)
	for _, item := range strings.Fields(lowered) {
		term := strings.TrimSpace(item)
		if term == "" {
			continue
		}
		if strings.HasPrefix(term, "+") && len(term) > 1 {
			requiredTerms = append(requiredTerms, strings.TrimPrefix(term, "+"))
			continue
		}
		optionalTerms = append(optionalTerms, term)
	}

	requiredTerms = util.NormalizeUniqueStringList(requiredTerms)
	optionalTerms = util.NormalizeUniqueStringList(optionalTerms)
	if len(requiredTerms) == 0 && len(optionalTerms) == 0 {
		return parsedToolSearchQuery{}, false
	}

	return parsedToolSearchQuery{
		Raw:           raw,
		RequiredTerms: requiredTerms,
		OptionalTerms: optionalTerms,
	}, true
}

// buildLocalRuntimeToolCandidates 预先拆分工具名，减少后续重复解析。
func buildLocalRuntimeToolCandidates(rows []dao.EnabledRuntimeToolSearchRow) []localRuntimeToolCandidate {
	candidates := make([]localRuntimeToolCandidate, 0, len(rows))
	for _, row := range rows {
		toolID := strings.TrimSpace(row.ToolID)
		toolName := strings.TrimSpace(row.ToolName)
		if toolID == "" || toolName == "" {
			continue
		}
		nameParts, fullName := parseSearchableToolName(toolName)
		candidates = append(candidates, localRuntimeToolCandidate{
			ToolID:            toolID,
			ToolName:          toolName,
			Description:       strings.ToLower(strings.TrimSpace(row.Description)),
			RuntimeType:       strings.ToLower(strings.TrimSpace(row.RuntimeType)),
			PluginName:        strings.ToLower(strings.TrimSpace(row.PluginName)),
			PluginDescription: strings.ToLower(strings.TrimSpace(row.PluginDescription)),
			NameParts:         nameParts,
			FullName:          fullName,
		})
	}
	return candidates
}

// parseSearchableToolName 对工具名执行下划线与 CamelCase 拆词。
func parseSearchableToolName(name string) ([]string, string) {
	normalized := camelCaseBoundaryPattern.ReplaceAllString(strings.TrimSpace(name), "$1 $2")
	normalized = strings.ReplaceAll(normalized, "_", " ")
	normalized = strings.ToLower(strings.TrimSpace(normalized))
	if normalized == "" {
		return []string{}, ""
	}

	parts := util.NormalizeUniqueStringList(strings.Fields(normalized))
	return parts, strings.Join(parts, " ")
}

// selectRuntimeToolsByName 处理 `select:tool_a,tool_b` 形式的精确选择。
func selectRuntimeToolsByName(candidates []localRuntimeToolCandidate, selectedNames []string, topK int, maxMCPTools int) []RuntimeToolSearchMatch {
	candidateByLowerName := make(map[string]localRuntimeToolCandidate, len(candidates))
	for _, candidate := range candidates {
		candidateByLowerName[strings.ToLower(candidate.ToolName)] = candidate
	}

	selected := make([]localRuntimeToolCandidate, 0, len(selectedNames))
	for _, name := range selectedNames {
		candidate, ok := candidateByLowerName[strings.ToLower(strings.TrimSpace(name))]
		if !ok {
			continue
		}
		selected = append(selected, candidate)
	}
	return finalizeRuntimeToolMatches(selected, topK, maxMCPTools)
}

// searchRuntimeToolsByKeywords 处理普通关键词与 `+required optional` 搜索。
func searchRuntimeToolsByKeywords(candidates []localRuntimeToolCandidate, parsed parsedToolSearchQuery, topK int, maxMCPTools int) []RuntimeToolSearchMatch {
	scoringTerms := append([]string{}, parsed.RequiredTerms...)
	if len(scoringTerms) == 0 {
		scoringTerms = append(scoringTerms, parsed.OptionalTerms...)
	} else {
		scoringTerms = append(scoringTerms, parsed.OptionalTerms...)
	}
	scoringTerms = util.NormalizeUniqueStringList(scoringTerms)
	if len(scoringTerms) == 0 {
		return []RuntimeToolSearchMatch{}
	}

	scored := make([]scoredToolCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if !candidateMatchesAllRequiredTerms(candidate, parsed.RequiredTerms) {
			continue
		}
		score := scoreRuntimeToolCandidate(candidate, scoringTerms)
		if score <= 0 {
			continue
		}
		scored = append(scored, scoredToolCandidate{
			Candidate: candidate,
			Score:     score,
		})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			return scored[i].Candidate.ToolName < scored[j].Candidate.ToolName
		}
		return scored[i].Score > scored[j].Score
	})

	ordered := make([]localRuntimeToolCandidate, 0, len(scored))
	for _, item := range scored {
		ordered = append(ordered, item.Candidate)
	}
	return finalizeRuntimeToolMatches(ordered, topK, maxMCPTools)
}

// candidateMatchesAllRequiredTerms 保证 `+term` 语义为必选命中。
func candidateMatchesAllRequiredTerms(candidate localRuntimeToolCandidate, requiredTerms []string) bool {
	for _, term := range requiredTerms {
		if !candidateContainsTerm(candidate, term) {
			return false
		}
	}
	return true
}

// scoreRuntimeToolCandidate 使用 Claude Code 风格的局部加权规则打分。
func scoreRuntimeToolCandidate(candidate localRuntimeToolCandidate, terms []string) int {
	score := 0
	for _, term := range terms {
		term = strings.ToLower(strings.TrimSpace(term))
		if term == "" {
			continue
		}

		if containsExactNamePart(candidate.NameParts, term) {
			score += 10
		} else if containsPartialNamePart(candidate.NameParts, term) {
			score += 5
		}

		if strings.Contains(candidate.FullName, term) {
			score += 3
		}
		if strings.Contains(candidate.Description, term) {
			score += 2
		}
		if strings.Contains(candidate.PluginName, term) {
			score += 4
		}
		if strings.Contains(candidate.PluginDescription, term) {
			score += 1
		}
	}
	return score
}

// candidateContainsTerm 为必选词过滤提供统一命中判定。
func candidateContainsTerm(candidate localRuntimeToolCandidate, term string) bool {
	term = strings.ToLower(strings.TrimSpace(term))
	if term == "" {
		return false
	}
	return containsExactNamePart(candidate.NameParts, term) ||
		containsPartialNamePart(candidate.NameParts, term) ||
		strings.Contains(candidate.FullName, term) ||
		strings.Contains(candidate.Description, term) ||
		strings.Contains(candidate.PluginName, term) ||
		strings.Contains(candidate.PluginDescription, term)
}

func containsExactNamePart(parts []string, term string) bool {
	for _, part := range parts {
		if part == term {
			return true
		}
	}
	return false
}

func containsPartialNamePart(parts []string, term string) bool {
	for _, part := range parts {
		if strings.Contains(part, term) {
			return true
		}
	}
	return false
}

// finalizeRuntimeToolMatches 统一处理去重、topK 截断与 MCP 数量限制。
func finalizeRuntimeToolMatches(candidates []localRuntimeToolCandidate, topK int, maxMCPTools int) []RuntimeToolSearchMatch {
	results := make([]RuntimeToolSearchMatch, 0, topK)
	seenToolIDs := make(map[string]struct{}, topK)
	mcpCount := 0
	for _, candidate := range candidates {
		if len(results) >= topK {
			break
		}
		if _, exists := seenToolIDs[candidate.ToolID]; exists {
			continue
		}
		if candidate.RuntimeType == pluginRuntimeMCP {
			if mcpCount >= maxMCPTools {
				continue
			}
			mcpCount++
		}
		seenToolIDs[candidate.ToolID] = struct{}{}
		results = append(results, RuntimeToolSearchMatch{
			ToolID:   candidate.ToolID,
			ToolName: candidate.ToolName,
		})
	}
	return results
}

// normalizeToolSearchTopK 归一化工具上限，允许调用方在 1-10 范围内显式收缩结果集。
func normalizeToolSearchTopK(value int) int {
	if value <= 0 {
		return defaultToolSearchTopK
	}
	if value > maxToolSearchTopK {
		return maxToolSearchTopK
	}
	return value
}

// normalizeToolSearchMaxMCPTools 归一化 mcp 限流阈值，并保证最小可用值。
func normalizeToolSearchMaxMCPTools(value int) int {
	if value <= 0 {
		return defaultToolSearchMaxMCPTools
	}
	return value
}

// normalizeToolSearchRuntimeTypes 归一化 runtime_type 白名单，默认包含 api/mcp/code。
func normalizeToolSearchRuntimeTypes(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case pluginRuntimeAPI, pluginRuntimeMCP, pluginRuntimeCode:
			normalized = append(normalized, strings.ToLower(strings.TrimSpace(value)))
		}
	}
	normalized = util.NormalizeUniqueStringList(normalized)
	if len(normalized) > 0 {
		return normalized
	}
	return []string{pluginRuntimeAPI, pluginRuntimeMCP, pluginRuntimeCode}
}
