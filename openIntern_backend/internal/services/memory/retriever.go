package memory

import (
	"context"
	"fmt"
	"strings"

	"openIntern/internal/services/memory/contracts"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
)

// MemoryRetrieverService is the facade used by agent runtime to inject long-term memory context.
type MemoryRetrieverService struct{}

// MemoryRetriever is the shared service singleton for long-term memory injection.
var MemoryRetriever = new(MemoryRetrieverService)

// Retrieve returns provider-agnostic memory snippets for the latest user input.
func (s *MemoryRetrieverService) Retrieve(ctx context.Context, userID string, inputText string) ([]contracts.RetrievedMemory, error) {
	return currentRetrieverBackend().Retrieve(ctx, userID, inputText)
}

// BuildContextMessage converts retrieved memories into one temporary system message for prompt injection.
func BuildContextMessage(memories []contracts.RetrievedMemory) *types.Message {
	content := BuildContextMessageContent(memories)
	if content == "" {
		return nil
	}
	return &types.Message{
		ID:      events.GenerateMessageID(),
		Role:    types.RoleSystem,
		Content: content,
	}
}

// BuildContextMessageContent formats provider-agnostic memory snippets into prompt text.
func BuildContextMessageContent(memories []contracts.RetrievedMemory) string {
	if len(memories) == 0 {
		return ""
	}
	lines := make([]string, 0, len(memories)+1)
	lines = append(lines, "以下是与当前请求相关的长期记忆，仅在确实相关时参考；若与用户当前明确要求冲突，以当前要求为准：")
	displayIndex := 1
	for _, item := range memories {
		content, ok := itemContent(item)
		if !ok {
			continue
		}
		lines = append(lines, contentLine(displayIndex, content))
		displayIndex++
	}
	if len(lines) == 1 {
		return ""
	}
	return joinLines(lines)
}

// itemContent normalizes one retrieved memory into prompt-ready content.
func itemContent(item contracts.RetrievedMemory) (string, bool) {
	content := strings.TrimSpace(item.Content)
	if content == "" {
		return "", false
	}
	return content, true
}

// contentLine formats one numbered memory line for the injected system prompt.
func contentLine(index int, content string) string {
	return fmt.Sprintf("%d. %s", index, content)
}

// joinLines keeps prompt assembly in one place for provider-agnostic memory snippets.
func joinLines(lines []string) string {
	return strings.Join(lines, "\n")
}
