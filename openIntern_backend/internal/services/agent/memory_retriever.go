package agent

import (
	"context"
	"strings"

	memorysvc "openIntern/internal/services/memory"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
)

// injectRetrievedMemoryContext retrieves memory snippets and injects them as one temporary system message.
func injectRetrievedMemoryContext(ctx context.Context, retriever MemoryRetriever, input *types.RunAgentInput) (*types.RunAgentInput, error) {
	if input == nil || retriever == nil {
		return input, nil
	}

	query := latestUserMessageText(input.Messages)
	if query == "" {
		return input, nil
	}
	memories, err := retriever.Retrieve(ctx, query)
	if err != nil {
		return input, err
	}
	message := memorysvc.BuildContextMessage(memories)
	if memoryMessageContentText(message) == "" {
		return input, nil
	}

	cloned := cloneRunAgentInputWithMessages(input, input.Messages)
	if cloned == nil {
		return input, nil
	}
	injectMessageBeforeUserAt(cloned, findLastUserMessageIndex(cloned.Messages), *message)
	return cloned, nil
}

// latestUserMessageText extracts the latest plain-text user message as the retrieval query.
func latestUserMessageText(messages []types.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != types.RoleUser {
			continue
		}
		text := strings.TrimSpace(memoryMessageContentText(&messages[i]))
		if text != "" {
			return text
		}
	}
	return ""
}

// memoryMessageContentText extracts the text form of the injected memory message content.
func memoryMessageContentText(message *types.Message) string {
	if message == nil {
		return ""
	}
	if value, ok := message.Content.(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}
