package dao

import (
	"context"
	"errors"
	"strings"

	"openIntern/internal/database"
)

// KnowledgeBaseSearchMatch 表示知识库检索返回的一条资源命中项。
type KnowledgeBaseSearchMatch struct {
	KnowledgeBaseName string
	URI               string
	Abstract          string
	Score             float64
	ContextType       string
	IsLeaf            bool
	Level             int
}

type knowledgeBaseSearchResult struct {
	Resources []knowledgeBaseSearchResource `json:"resources"`
}

type knowledgeBaseSearchResource struct {
	URI         string  `json:"uri"`
	Abstract    string  `json:"abstract"`
	Score       float64 `json:"score"`
	ContextType string  `json:"context_type"`
	IsLeaf      bool    `json:"is_leaf"`
	Level       int     `json:"level"`
}

// SearchInKnowledgeBases 在指定知识库范围内执行 search 检索并返回资源命中。
func (d *KnowledgeBaseDAO) SearchInKnowledgeBases(ctx context.Context, query string, kbNames []string, limitPerKB int) ([]KnowledgeBaseSearchMatch, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limitPerKB <= 0 {
		limitPerKB = 5
	}
	cleanNames := make([]string, 0, len(kbNames))
	seenName := make(map[string]struct{}, len(kbNames))
	for _, name := range kbNames {
		kbName, err := d.CleanName(name)
		if err != nil {
			continue
		}
		if _, exists := seenName[kbName]; exists {
			continue
		}
		seenName[kbName] = struct{}{}
		cleanNames = append(cleanNames, kbName)
	}
	if len(cleanNames) == 0 {
		return nil, nil
	}
	userID, err := OpenVikingUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	matches := make([]KnowledgeBaseSearchMatch, 0, len(cleanNames)*limitPerKB)
	seenURI := make(map[string]struct{}, len(cleanNames)*limitPerKB)
	for _, kbName := range cleanNames {
		kb, err := KBLocal.GetByName(ctx, userID, kbName)
		if err != nil {
			if errors.Is(err, ErrKBNotFound) {
				continue
			}
			return nil, err
		}
		targetURI := strings.TrimSpace(kb.OpenVikingURI)
		if targetURI == "" {
			targetURI, err = d.URI(ctx, kbName)
			if err != nil {
				return nil, err
			}
		}
		items, err := d.searchByTargetURI(ctx, query, targetURI, limitPerKB)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			key := strings.TrimSpace(item.URI)
			if key == "" {
				continue
			}
			if _, exists := seenURI[key]; exists {
				continue
			}
			seenURI[key] = struct{}{}
			item.KnowledgeBaseName = kbName
			matches = append(matches, item)
		}
	}
	return matches, nil
}

// searchByTargetURI 调用 OpenViking search API，在指定 URI 前缀内检索资源。
func (d *KnowledgeBaseDAO) searchByTargetURI(ctx context.Context, query string, targetURI string, limit int) ([]KnowledgeBaseSearchMatch, error) {
	if limit <= 0 {
		limit = 5
	}
	payload := map[string]any{
		"query":      query,
		"target_uri": strings.TrimSpace(targetURI),
		"limit":      limit,
	}
	body, err := database.Context.Post(ctx, "/api/v1/search/search", payload)
	if err != nil {
		return nil, err
	}
	var result knowledgeBaseSearchResult
	if err := decodeStoreResult(body, &result); err != nil {
		return nil, err
	}
	if len(result.Resources) == 0 {
		return []KnowledgeBaseSearchMatch{}, nil
	}
	matches := make([]KnowledgeBaseSearchMatch, 0, len(result.Resources))
	for _, item := range result.Resources {
		matches = append(matches, KnowledgeBaseSearchMatch{
			URI:         strings.TrimSpace(item.URI),
			Abstract:    strings.TrimSpace(item.Abstract),
			Score:       item.Score,
			ContextType: strings.TrimSpace(item.ContextType),
			IsLeaf:      item.IsLeaf,
			Level:       item.Level,
		})
	}
	return matches, nil
}
