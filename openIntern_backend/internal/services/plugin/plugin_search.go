package plugin

import (
	"context"
	"strings"

	"openIntern/internal/dao"
	"openIntern/internal/database"
	"openIntern/internal/util"
)

const (
	defaultToolSearchTopK               = 10
	maxToolSearchTopK                   = 10
	defaultToolSearchMaxMCPTools        = 3
	defaultToolSearchCandidateFactor    = 4
	defaultToolSearchTargetURI          = "viking://resources/tools/"
	defaultToolSearchMinCandidateAmount = 20
	defaultToolSearchScoreThreshold     = 0
)

// ToolSearchOptions 表示 search 模式下工具召回参数。
type ToolSearchOptions struct {
	TopK         int
	RuntimeTypes []string
	MinScore     float64
	MaxMCPTools  int
	TargetURI    string
}

// RuntimeToolSearchMatch 表示语义检索命中的最小工具元信息。
type RuntimeToolSearchMatch struct {
	ToolID   string
	ToolName string
}

// SearchRuntimeToolIDs 使用 OpenViking find 召回工具并执行 MySQL 启用态二次过滤。
func (s *PluginService) SearchRuntimeToolIDs(ctx context.Context, query string, options ToolSearchOptions) ([]string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []string{}, nil
	}

	topK := normalizeToolSearchTopK(options.TopK)
	maxMCPTools := normalizeToolSearchMaxMCPTools(options.MaxMCPTools)
	scoreThreshold := normalizeToolSearchScoreThreshold(options.MinScore)
	runtimeTypes := normalizeToolSearchRuntimeTypes(options.RuntimeTypes)
	targetURI := resolveToolSearchTargetURI(options.TargetURI)

	candidateLimit := topK * defaultToolSearchCandidateFactor
	if candidateLimit < defaultToolSearchMinCandidateAmount {
		candidateLimit = defaultToolSearchMinCandidateAmount
	}

	matches, err := dao.Plugin.FindToolSearchMatches(ctx, dao.PluginToolSearchFilter{
		Query:          query,
		TargetURI:      targetURI,
		Limit:          candidateLimit,
		ScoreThreshold: scoreThreshold,
	})
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return []string{}, nil
	}

	candidateToolIDs := make([]string, 0, len(matches))
	for _, item := range matches {
		candidateToolIDs = append(candidateToolIDs, item.ToolID)
	}
	candidateToolIDs = util.NormalizeUniqueStringList(candidateToolIDs)
	if len(candidateToolIDs) == 0 {
		return []string{}, nil
	}

	enabledRecords, err := dao.Plugin.ListEnabledRuntimeToolRecords(candidateToolIDs, runtimeTypes)
	if err != nil {
		return nil, err
	}
	if len(enabledRecords) == 0 {
		return []string{}, nil
	}

	runtimeByToolID := make(map[string]string, len(enabledRecords))
	for _, item := range enabledRecords {
		runtimeByToolID[item.ToolID] = strings.ToLower(strings.TrimSpace(item.RuntimeType))
	}

	selected := make([]string, 0, topK)
	selectedSet := make(map[string]struct{}, topK)
	mcpCount := 0
	for _, item := range matches {
		if len(selected) >= topK {
			break
		}
		toolID := strings.TrimSpace(item.ToolID)
		if toolID == "" {
			continue
		}
		if _, exists := selectedSet[toolID]; exists {
			continue
		}
		runtimeType, ok := runtimeByToolID[toolID]
		if !ok {
			continue
		}
		if runtimeType == pluginRuntimeMCP {
			if mcpCount >= maxMCPTools {
				continue
			}
			mcpCount++
		}
		selectedSet[toolID] = struct{}{}
		selected = append(selected, toolID)
	}

	return selected, nil
}

// SearchRuntimeTools 返回保序的命中工具列表，供运行时 tool_search 元工具复用。
func (s *PluginService) SearchRuntimeTools(ctx context.Context, query string, options ToolSearchOptions) ([]RuntimeToolSearchMatch, error) {
	selectedToolIDs, err := s.SearchRuntimeToolIDs(ctx, query, options)
	if err != nil {
		return nil, err
	}
	if len(selectedToolIDs) == 0 {
		return []RuntimeToolSearchMatch{}, nil
	}

	toolNameByID, err := s.loadEnabledRuntimeToolNameMap(selectedToolIDs)
	if err != nil {
		return nil, err
	}
	if len(toolNameByID) == 0 {
		return []RuntimeToolSearchMatch{}, nil
	}

	matches := make([]RuntimeToolSearchMatch, 0, len(selectedToolIDs))
	for _, toolID := range selectedToolIDs {
		toolName := strings.TrimSpace(toolNameByID[toolID])
		if toolName == "" {
			continue
		}
		matches = append(matches, RuntimeToolSearchMatch{
			ToolID:   toolID,
			ToolName: toolName,
		})
	}
	return matches, nil
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

// normalizeToolSearchScoreThreshold 归一化分数阈值，默认不主动下发阈值给 OpenViking。
func normalizeToolSearchScoreThreshold(value float64) float64 {
	if value <= 0 {
		return defaultToolSearchScoreThreshold
	}
	if value > 1 {
		return 1
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

// resolveToolSearchTargetURI 解析工具索引根 URI，优先使用调用参数，再读取 OpenViking 配置。
func resolveToolSearchTargetURI(targetURI string) string {
	if resolved := normalizeToolSearchTargetURI(targetURI); resolved != "" {
		return resolved
	}
	if database.Context != nil {
		if resolved := normalizeToolSearchTargetURI(database.Context.ToolsRoot()); resolved != "" {
			return resolved
		}
	}
	return defaultToolSearchTargetURI
}

// normalizeToolSearchTargetURI 规范化目标 URI，统一为带尾斜杠形式。
func normalizeToolSearchTargetURI(targetURI string) string {
	targetURI = strings.TrimSpace(targetURI)
	if targetURI == "" {
		return ""
	}
	return strings.TrimRight(targetURI, "/") + "/"
}
