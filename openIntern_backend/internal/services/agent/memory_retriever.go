package agent

import (
	"context"
	"strings"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
)

// injectRetrievedMemoryContext asks the retriever for a temporary memory message and places it before the latest user turn.
func injectRetrievedMemoryContext(ctx context.Context, retriever MemoryRetriever, input *types.RunAgentInput) (*types.RunAgentInput, []string, error) {
	if input == nil || retriever == nil {
		return input, nil, nil
	}

	message, uris, err := retriever.BuildMemoryContext(ctx, input)
	if err != nil {
		return input, nil, err
	}
	if memoryMessageContentText(message) == "" {
		return input, nil, nil
	}

	cloned := cloneRunAgentInputWithMessages(input, input.Messages)
	if cloned == nil {
		return input, nil, nil
	}
	injectMessageBeforeUserAt(cloned, findLastUserMessageIndex(cloned.Messages), *message)
	return cloned, uris, nil
}

// recordRetrievedMemoryUsage persists the memory URIs that were actually injected for this run.
func recordRetrievedMemoryUsage(store MemoryUsageLogStore, threadID, runID string, uris []string) error {
	if store == nil || len(uris) == 0 {
		return nil
	}
	return store.RecordRunMemoryUsage(threadID, runID, uris)
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
