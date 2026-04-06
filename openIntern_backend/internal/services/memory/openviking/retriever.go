package openviking

import (
	"context"
	"sort"
	"strings"
	"time"

	"openIntern/internal/config"
	"openIntern/internal/dao"
	"openIntern/internal/services/memory/contracts"
)

const (
	defaultUserMemoryFindLimit   = 4
	defaultAgentMemoryFindLimit  = 2
	defaultMemoryContextMaxItems = 5
	// defaultMemoryScoreThreshold filters low-relevance long-term memory hits before prompt injection.
	defaultMemoryScoreThreshold = 0.4
	defaultMemorySearchTimeout  = 2 * time.Second
)

// Retriever implements OpenViking-backed memory retrieval.
type Retriever struct {
	searchTimeout time.Duration
}

// NewRetriever builds one OpenViking retrieval backend from configuration.
func NewRetriever(cfg config.OpenVikingConfig) *Retriever {
	return &Retriever{
		searchTimeout: durationFromSeconds(cfg.MemorySearchTimeoutSeconds, defaultMemorySearchTimeout),
	}
}

// Configured reports whether OpenViking retrieval is available.
func (r *Retriever) Configured() bool {
	return dao.MemorySearch.Configured()
}

// Retrieve returns provider-agnostic memory snippets for the provided latest user input text.
func (r *Retriever) Retrieve(ctx context.Context, userID string, inputText string) ([]contracts.RetrievedMemory, error) {
	if !r.Configured() {
		return nil, nil
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}

	query := strings.TrimSpace(inputText)
	if query == "" {
		return nil, nil
	}

	searchCtx := ctx
	if searchCtx == nil {
		searchCtx = context.Background()
	}
	if _, hasDeadline := searchCtx.Deadline(); !hasDeadline && r.searchTimeout > 0 {
		var cancel context.CancelFunc
		searchCtx, cancel = context.WithTimeout(searchCtx, r.searchTimeout)
		defer cancel()
	}

	matches, err := r.findRelevantMemoryMatches(searchCtx, userID, query)
	if err != nil {
		return nil, err
	}
	return toRetrievedMemories(matches), nil
}

// findRelevantMemoryMatches queries user memories first and agent memories second, then trims the merged result set.
func (r *Retriever) findRelevantMemoryMatches(ctx context.Context, userID string, query string) ([]dao.MemorySearchMatch, error) {
	userMatches, err := dao.MemorySearch.FindMemoryMatches(ctx, dao.MemorySearchFilter{
		Query:          query,
		TargetURI:      dao.MemorySearch.UserRootURI(userID),
		Limit:          defaultUserMemoryFindLimit,
		ScoreThreshold: defaultMemoryScoreThreshold,
	})
	if err != nil {
		return nil, err
	}

	agentMatches, err := dao.MemorySearch.FindMemoryMatches(ctx, dao.MemorySearchFilter{
		Query:          query,
		TargetURI:      dao.MemorySearch.AgentRootURI(),
		Limit:          defaultAgentMemoryFindLimit,
		ScoreThreshold: defaultMemoryScoreThreshold,
	})
	if err != nil {
		return nil, err
	}

	return mergeMemoryMatches(userMatches, agentMatches, defaultMemoryContextMaxItems), nil
}

// mergeMemoryMatches preserves user-memory priority, sorts within each source by score, and removes duplicate URIs.
func mergeMemoryMatches(userMatches []dao.MemorySearchMatch, agentMatches []dao.MemorySearchMatch, limit int) []dao.MemorySearchMatch {
	if limit <= 0 {
		limit = defaultMemoryContextMaxItems
	}

	ordered := make([]dao.MemorySearchMatch, 0, len(userMatches)+len(agentMatches))
	ordered = append(ordered, sortMemoryMatches(userMatches)...)
	ordered = append(ordered, sortMemoryMatches(agentMatches)...)

	merged := make([]dao.MemorySearchMatch, 0, minInt(limit, len(ordered)))
	seen := make(map[string]struct{}, len(ordered))
	for _, item := range ordered {
		uri := normalizeMemoryMatchURI(item.URI)
		if uri == "" {
			continue
		}
		if _, exists := seen[uri]; exists {
			continue
		}
		seen[uri] = struct{}{}
		item.URI = uri
		merged = append(merged, item)
		if len(merged) >= limit {
			break
		}
	}
	return merged
}

// sortMemoryMatches orders a single memory source by descending relevance score.
func sortMemoryMatches(matches []dao.MemorySearchMatch) []dao.MemorySearchMatch {
	cloned := append([]dao.MemorySearchMatch(nil), matches...)
	sort.SliceStable(cloned, func(i, j int) bool {
		return cloned[i].Score > cloned[j].Score
	})
	return cloned
}

// toRetrievedMemories drops provider-specific metadata and keeps only content plus score.
func toRetrievedMemories(matches []dao.MemorySearchMatch) []contracts.RetrievedMemory {
	memories := make([]contracts.RetrievedMemory, 0, len(matches))
	for _, item := range matches {
		content := strings.TrimSpace(item.Abstract)
		if content == "" {
			continue
		}
		memories = append(memories, contracts.RetrievedMemory{
			Content: content,
			Score:   item.Score,
		})
	}
	return memories
}

// normalizeMemoryMatchURI trims query and fragment suffixes so duplicate provider hits collapse correctly.
func normalizeMemoryMatchURI(uri string) string {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return ""
	}
	if idx := strings.Index(uri, "?"); idx >= 0 {
		uri = uri[:idx]
	}
	if idx := strings.Index(uri, "#"); idx >= 0 {
		uri = uri[:idx]
	}
	return strings.TrimRight(strings.TrimSpace(uri), "/")
}

// minInt returns the smaller integer, and keeps slice preallocation predictable.
func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

// durationFromSeconds converts a positive seconds value into a duration or returns the fallback.
func durationFromSeconds(seconds int, fallback time.Duration) time.Duration {
	if seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}
