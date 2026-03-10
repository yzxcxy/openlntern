package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"openIntern/internal/config"
	"openIntern/internal/dao"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
)

const (
	defaultUserMemoryFindLimit   = 4
	defaultAgentMemoryFindLimit  = 2
	defaultPreferenceFindLimit   = 2
	defaultMemoryContextMaxItems = 5
	// defaultMemoryScoreThreshold filters low-relevance long-term memory hits before prompt injection.
	defaultMemoryScoreThreshold = 0.4
	defaultMemorySearchTimeout  = 2 * time.Second
	defaultPreferenceQuery      = "用户偏好 回答风格 语气 风格偏好 写作风格"
	memoryContextPrefix         = "以下是与当前请求相关的长期记忆，仅在确实相关时参考；若与用户当前明确要求冲突，以当前要求为准："
)

// MemoryRetrieverService retrieves OpenViking long-term memories for the current turn.
type MemoryRetrieverService struct{}

// MemoryRetriever is the shared service singleton for long-term memory injection.
var MemoryRetriever = new(MemoryRetrieverService)
var memorySearchTimeout = defaultMemorySearchTimeout

// BuildMemoryContext builds one temporary system message from the current turn's most relevant memories.
func (s *MemoryRetrieverService) BuildMemoryContext(ctx context.Context, input *types.RunAgentInput) (*types.Message, []string, error) {
	if input == nil || !dao.MemorySearch.Configured() {
		return nil, nil, nil
	}

	query := extractMemoryQuery(input.Messages)
	if query == "" {
		return nil, nil, nil
	}

	searchCtx := ctx
	if searchCtx == nil {
		searchCtx = context.Background()
	}
	if _, hasDeadline := searchCtx.Deadline(); !hasDeadline && memorySearchTimeout > 0 {
		var cancel context.CancelFunc
		searchCtx, cancel = context.WithTimeout(searchCtx, memorySearchTimeout)
		defer cancel()
	}

	matches, err := s.findRelevantMemoryMatches(searchCtx, query)
	if err != nil {
		return nil, nil, err
	}
	content := buildMemoryContextMessage(matches)
	if content == "" {
		return nil, nil, nil
	}
	uris := collectMemoryMatchURIs(matches)

	return &types.Message{
		ID:      events.GenerateMessageID(),
		Role:    types.RoleSystem,
		Content: content,
	}, uris, nil
}

// InitMemoryRetriever loads the retrieval timeout used before the model run starts.
func InitMemoryRetriever(cfg config.OpenVikingConfig) {
	memorySearchTimeout = memoryRetrieverDurationFromSeconds(cfg.MemorySearchTimeoutSeconds, defaultMemorySearchTimeout)
}

// findRelevantMemoryMatches queries user memories first and agent memories second, then trims the merged result set.
func (s *MemoryRetrieverService) findRelevantMemoryMatches(ctx context.Context, query string) ([]dao.MemorySearchMatch, error) {
	userMatches, err := dao.MemorySearch.FindMemoryMatches(ctx, dao.MemorySearchFilter{
		Query:          query,
		TargetURI:      dao.MemorySearch.UserRootURI(),
		Limit:          defaultUserMemoryFindLimit,
		ScoreThreshold: defaultMemoryScoreThreshold,
	})
	if err != nil {
		return nil, err
	}

	if !containsPreferenceMemory(userMatches) {
		preferenceMatches, preferenceErr := dao.MemorySearch.FindMemoryMatches(ctx, dao.MemorySearchFilter{
			Query:          defaultPreferenceQuery,
			TargetURI:      dao.MemorySearch.UserRootURI(),
			Limit:          defaultPreferenceFindLimit,
			ScoreThreshold: defaultMemoryScoreThreshold,
		})
		if preferenceErr == nil {
			userMatches = append(userMatches, preferenceMatches...)
		}
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

// containsPreferenceMemory reports whether the current match set already includes stable user-level style or preference guidance.
func containsPreferenceMemory(matches []dao.MemorySearchMatch) bool {
	for _, item := range matches {
		switch strings.ToLower(strings.TrimSpace(item.MemoryType)) {
		case "preferences", "profile":
			return true
		}
	}
	return false
}

// extractMemoryQuery reuses the latest user text as the retrieval query for long-term memory.
func extractMemoryQuery(messages []types.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != types.RoleUser {
			continue
		}
		text := strings.TrimSpace(extractMessageText(messages[i]))
		if text != "" {
			return text
		}
	}
	return ""
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

// buildMemoryContextMessage formats a small memory pack for prompt injection.
func buildMemoryContextMessage(matches []dao.MemorySearchMatch) string {
	if len(matches) == 0 {
		return ""
	}

	lines := make([]string, 0, len(matches)+1)
	lines = append(lines, memoryContextPrefix)
	displayIndex := 1
	for _, item := range matches {
		// 仅注入摘要文本，避免把 URI/类型等检索元数据暴露给模型提示词。
		summary := strings.TrimSpace(item.Abstract)
		if summary == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%d. %s", displayIndex, summary))
		displayIndex++
	}

	if len(lines) == 1 {
		return ""
	}
	return strings.Join(lines, "\n")
}

// collectMemoryMatchURIs returns the injected memory URIs in prompt order for later used(contexts) reporting.
func collectMemoryMatchURIs(matches []dao.MemorySearchMatch) []string {
	uris := make([]string, 0, len(matches))
	for _, item := range matches {
		uri := normalizeMemoryMatchURI(item.URI)
		if uri == "" {
			continue
		}
		uris = append(uris, uri)
	}
	return uris
}

// normalizeMemoryMatchURI trims query and fragment suffixes so duplicate hits collapse correctly.
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

// memoryRetrieverDurationFromSeconds converts a positive seconds value into a duration or returns the fallback.
func memoryRetrieverDurationFromSeconds(seconds int, fallback time.Duration) time.Duration {
	if seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}
