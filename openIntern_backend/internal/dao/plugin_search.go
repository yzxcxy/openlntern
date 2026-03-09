package dao

import (
	"context"
	"path"
	"strings"

	"openIntern/internal/database"
)

// PluginToolSearchFilter 描述工具召回查询条件。
type PluginToolSearchFilter struct {
	Query          string
	TargetURI      string
	Limit          int
	ScoreThreshold float64
}

// PluginToolSearchMatch 表示 OpenViking 召回的单条工具命中。
type PluginToolSearchMatch struct {
	ToolID string
	URI    string
	Score  float64
}

type openVikingFindResult struct {
	Resources []openVikingFindResource `json:"resources"`
}

type openVikingFindResource struct {
	URI   string  `json:"uri"`
	Score float64 `json:"score"`
}

// FindToolSearchMatches 使用 OpenViking find 方法在目标 URI 下召回候选工具。
func (d *PluginDAO) FindToolSearchMatches(ctx context.Context, filter PluginToolSearchFilter) ([]PluginToolSearchMatch, error) {
	if !contextStoreReady() {
		return nil, nil
	}

	query := strings.TrimSpace(filter.Query)
	if query == "" {
		return nil, nil
	}

	targetURI := strings.TrimSpace(filter.TargetURI)
	if targetURI == "" {
		return nil, nil
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	payload := map[string]any{
		"query":      query,
		"target_uri": targetURI,
		"limit":      limit,
	}
	if filter.ScoreThreshold > 0 {
		payload["score_threshold"] = filter.ScoreThreshold
	}

	body, err := database.Context.Post(ctx, "/api/v1/search/find", payload)
	if err != nil {
		return nil, err
	}

	var result openVikingFindResult
	if err := decodeStoreResult(body, &result); err != nil {
		return nil, err
	}
	if len(result.Resources) == 0 {
		return []PluginToolSearchMatch{}, nil
	}

	seen := make(map[string]struct{}, len(result.Resources))
	matches := make([]PluginToolSearchMatch, 0, len(result.Resources))
	for _, item := range result.Resources {
		uri := strings.TrimSpace(item.URI)
		if !isToolSearchURIUnderTarget(uri, targetURI) {
			continue
		}
		toolID := extractToolIDFromVikingURIWithTarget(uri, targetURI)
		if toolID == "" {
			continue
		}
		if _, exists := seen[toolID]; exists {
			continue
		}
		seen[toolID] = struct{}{}
		matches = append(matches, PluginToolSearchMatch{
			ToolID: toolID,
			URI:    uri,
			Score:  item.Score,
		})
	}
	return matches, nil
}

// extractToolIDFromVikingURIWithTarget 基于 target_uri 和 URI 路径层级提取 tool_id。
func extractToolIDFromVikingURIWithTarget(uri string, targetURI string) string {
	normalizedURI := normalizeVikingURI(uri)
	normalizedTarget := normalizeVikingURI(targetURI)
	if normalizedURI == "" || normalizedTarget == "" {
		return extractToolIDFromVikingURI(uri)
	}
	normalizedTarget = strings.TrimRight(normalizedTarget, "/") + "/"
	if !strings.HasPrefix(normalizedURI, normalizedTarget) {
		return ""
	}

	relative := strings.TrimPrefix(normalizedURI, normalizedTarget)
	parts := splitPathSegments(relative)
	if len(parts) < 2 {
		return ""
	}

	base := path.Base(normalizedURI)
	// 插件级目录摘要（例如 tools/<plugin_id>/.overview.md）不属于具体工具，直接忽略。
	if strings.HasPrefix(base, ".") && len(parts) < 3 {
		return ""
	}
	return extractToolIDFromVikingURI(normalizedURI)
}

// extractToolIDFromVikingURI 从 viking URI 的最后路径段推导 tool_id。
func extractToolIDFromVikingURI(uri string) string {
	normalized := normalizeVikingURI(uri)
	if normalized == "" {
		return ""
	}

	base := path.Base(normalized)
	if base == "." || base == "/" || base == "" {
		return ""
	}

	// OpenViking 会返回目录级摘要资源（例如 `/.overview.md`），这类 URI 需要回退父目录名作为 tool_id。
	if strings.HasPrefix(base, ".") {
		return toolIDFromParentDir(normalized)
	}

	base = strings.TrimSuffix(base, path.Ext(base))
	base = strings.TrimSpace(base)
	if base == "" || strings.HasPrefix(base, ".") {
		return toolIDFromParentDir(normalized)
	}
	return base
}

// isToolSearchURIUnderTarget 判断候选 URI 是否位于目标 URI 前缀下。
func isToolSearchURIUnderTarget(uri string, targetURI string) bool {
	candidate := normalizeVikingURI(uri)
	target := normalizeVikingURI(targetURI)
	if candidate == "" || target == "" {
		return false
	}
	target = strings.TrimRight(target, "/") + "/"
	return strings.HasPrefix(candidate, target)
}

// normalizeVikingURI 清理 URI 的 query/fragment，并去掉末尾斜杠。
func normalizeVikingURI(uri string) string {
	trimmed := strings.TrimSpace(uri)
	if trimmed == "" {
		return ""
	}
	if idx := strings.Index(trimmed, "?"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	if idx := strings.Index(trimmed, "#"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	return strings.TrimRight(strings.TrimSpace(trimmed), "/")
}

// splitPathSegments 将相对路径按 `/` 切分并移除空段。
func splitPathSegments(relative string) []string {
	relative = strings.TrimSpace(relative)
	if relative == "" {
		return []string{}
	}
	rawParts := strings.Split(relative, "/")
	parts := make([]string, 0, len(rawParts))
	for _, item := range rawParts {
		segment := strings.TrimSpace(item)
		if segment == "" {
			continue
		}
		parts = append(parts, segment)
	}
	return parts
}

// toolIDFromParentDir 使用 URI 父目录名推导 tool_id。
func toolIDFromParentDir(uri string) string {
	parent := strings.TrimSpace(path.Base(path.Dir(uri)))
	if parent == "" || parent == "." || parent == "/" {
		return ""
	}
	if strings.HasPrefix(parent, ".") {
		return ""
	}
	return parent
}
