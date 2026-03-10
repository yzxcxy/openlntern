package dao

import (
	"context"
	"strings"

	"openIntern/internal/database"
)

const (
	defaultUserMemoryRootURI  = "viking://user/default/memories/"
	defaultAgentMemoryRootURI = "viking://agent/default/memories/"
)

type openVikingMatchedContext struct {
	URI         string  `json:"uri"`
	ContextType string  `json:"context_type"`
	IsLeaf      bool    `json:"is_leaf"`
	Abstract    string  `json:"abstract"`
	Score       float64 `json:"score"`
}

type openVikingMemoryFindResult struct {
	Memories  []openVikingMatchedContext `json:"memories"`
	Resources []openVikingMatchedContext `json:"resources"`
}

// MemorySearchFilter describes one OpenViking memory retrieval request.
type MemorySearchFilter struct {
	Query          string
	TargetURI      string
	Limit          int
	ScoreThreshold float64
}

// MemorySearchMatch represents one OpenViking long-term memory hit.
type MemorySearchMatch struct {
	URI         string
	Abstract    string
	ContextType string
	MemoryType  string
	Score       float64
	IsLeaf      bool
}

// MemorySearchDAO provides OpenViking-backed long-term memory retrieval.
type MemorySearchDAO struct{}

// MemorySearch is the shared DAO singleton for long-term memory retrieval.
var MemorySearch = new(MemorySearchDAO)

// Configured reports whether OpenViking retrieval is available.
func (d *MemorySearchDAO) Configured() bool {
	return contextStoreReady()
}

// UserRootURI returns the fixed OpenViking root used for user memories.
func (d *MemorySearchDAO) UserRootURI() string {
	return defaultUserMemoryRootURI
}

// AgentRootURI returns the fixed OpenViking root used for agent memories.
func (d *MemorySearchDAO) AgentRootURI() string {
	return defaultAgentMemoryRootURI
}

// FindMemoryMatches searches one OpenViking memory subtree with the provided query.
func (d *MemorySearchDAO) FindMemoryMatches(ctx context.Context, filter MemorySearchFilter) ([]MemorySearchMatch, error) {
	if !d.Configured() {
		return nil, nil
	}

	query := strings.TrimSpace(filter.Query)
	if query == "" {
		return []MemorySearchMatch{}, nil
	}

	targetURI := normalizeMemoryTargetURI(filter.TargetURI)
	if targetURI == "" {
		return []MemorySearchMatch{}, nil
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 10
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

	var result openVikingMemoryFindResult
	if err := decodeStoreResult(body, &result); err != nil {
		return nil, err
	}

	candidates := pickMemoryContexts(result)
	if len(candidates) == 0 {
		return []MemorySearchMatch{}, nil
	}

	seen := make(map[string]struct{}, len(candidates))
	matches := make([]MemorySearchMatch, 0, len(candidates))
	for _, item := range candidates {
		uri := normalizeVikingURI(item.URI)
		if !isMemoryURIUnderTarget(uri, targetURI) {
			continue
		}
		if _, exists := seen[uri]; exists {
			continue
		}
		seen[uri] = struct{}{}
		matches = append(matches, MemorySearchMatch{
			URI:         uri,
			Abstract:    strings.TrimSpace(item.Abstract),
			ContextType: strings.TrimSpace(item.ContextType),
			MemoryType:  deriveMemoryTypeFromURI(uri),
			Score:       item.Score,
			IsLeaf:      item.IsLeaf,
		})
	}
	return matches, nil
}

// normalizeMemoryTargetURI normalizes a memory root URI to the trailing-slash form expected by OpenViking.
func normalizeMemoryTargetURI(targetURI string) string {
	targetURI = strings.TrimSpace(targetURI)
	if targetURI == "" {
		return ""
	}
	return strings.TrimRight(targetURI, "/") + "/"
}

// pickMemoryContexts prefers the dedicated memories field and falls back to resources for compatibility.
func pickMemoryContexts(result openVikingMemoryFindResult) []openVikingMatchedContext {
	if len(result.Memories) > 0 {
		return result.Memories
	}
	return result.Resources
}

// isMemoryURIUnderTarget verifies the candidate memory URI remains inside the requested root.
func isMemoryURIUnderTarget(uri string, targetURI string) bool {
	candidate := normalizeVikingURI(uri)
	target := normalizeMemoryTargetURI(targetURI)
	if candidate == "" || target == "" {
		return false
	}
	return strings.HasPrefix(candidate, strings.TrimRight(target, "/")+"/")
}

// deriveMemoryTypeFromURI infers the OpenViking memory category from the resource URI.
func deriveMemoryTypeFromURI(uri string) string {
	normalized := normalizeVikingURI(uri)
	switch {
	case normalized == "":
		return ""
	case strings.Contains(normalized, "/preferences/"):
		return "preferences"
	case strings.Contains(normalized, "/entities/"):
		return "entities"
	case strings.Contains(normalized, "/events/"):
		return "events"
	case strings.Contains(normalized, "/cases/"):
		return "cases"
	case strings.Contains(normalized, "/patterns/"):
		return "patterns"
	case strings.HasSuffix(normalized, "/.overview.md"), strings.HasSuffix(normalized, "/profile.md"):
		return "profile"
	default:
		return ""
	}
}
