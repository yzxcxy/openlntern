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
		toolID := extractToolIDFromVikingURI(item.URI)
		if toolID == "" {
			continue
		}
		if _, exists := seen[toolID]; exists {
			continue
		}
		seen[toolID] = struct{}{}
		matches = append(matches, PluginToolSearchMatch{
			ToolID: toolID,
			URI:    strings.TrimSpace(item.URI),
			Score:  item.Score,
		})
	}
	return matches, nil
}

// extractToolIDFromVikingURI 从 viking URI 的最后路径段推导 tool_id。
func extractToolIDFromVikingURI(uri string) string {
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
	trimmed = strings.TrimRight(trimmed, "/")
	if trimmed == "" {
		return ""
	}

	base := path.Base(trimmed)
	if base == "." || base == "/" {
		return ""
	}
	ext := path.Ext(base)
	if ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	return strings.TrimSpace(base)
}
