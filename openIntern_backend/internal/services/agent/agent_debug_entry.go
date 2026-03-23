package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"openIntern/internal/services/agent/agui"
	builtinTool "openIntern/internal/services/builtin_tool"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/cloudwego/eino/schema"
)

const (
	forwardedPropKeyDebugDefinition      = "debugDefinition"
	forwardedPropKeyDebugRuntimeOverride = "debugRuntimeOverride"
)

// RunDebugAgent executes one transient agent definition for editor-side testing without persistence.
func (s *Service) RunDebugAgent(ctx context.Context, w io.Writer, input *types.RunAgentInput) error {
	if input == nil {
		return nil
	}
	ownerID := ownerIDFromContext(ctx)
	if ownerID == "" {
		return fmt.Errorf("owner_id is required")
	}
	debugDetail, runtimeConfig, err := buildDebugAgentRequest(ctx, ownerID, input)
	if err != nil {
		return err
	}
	if len(input.Messages) == 0 {
		return fmt.Errorf("messages are required")
	}

	state := s.snapshotState()
	threadID := input.ThreadID
	if threadID == "" {
		threadID = events.GenerateThreadID()
	}
	acc := agui.NewAccumulator(threadID)
	baseSender := agui.NewSenderWithThreadID(ctx, w, threadID)
	sender := agui.NewAccumulatingSender(baseSender, acc)
	runID := baseSender.RunID()
	if runID == "" {
		return fmt.Errorf("run_id is required")
	}
	if err := sender.Start(); err != nil {
		log.Printf("RunDebugAgent start failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}

	compressedInput, compressionStats, err := s.compressInputContext(ctx, input, runtimeConfig, state)
	if err != nil {
		_ = sender.Error(err.Error(), "context_compress_failed")
		return err
	}
	if compressionStats != nil && compressionStats.Enabled && compressionStats.Triggered {
		log.Printf(
			"RunDebugAgent context compressed thread_id=%s run_id=%s original_tokens=%d compressed_tokens=%d removed_messages=%d soft_limit=%d hard_limit=%d summary_used=%t summary_updated=%t snapshot_index=%d",
			threadID,
			runID,
			compressionStats.OriginalTokens,
			compressionStats.CompressedTokens,
			compressionStats.RemovedMessages,
			compressionStats.SoftLimitTokens,
			compressionStats.HardLimitTokens,
			compressionStats.SummaryUsed,
			compressionStats.SummaryUpdated,
			compressionStats.SnapshotIndex,
		)
	}

	einoMessages, err := agui.AGUIRunInputToEinoMessages(compressedInput)
	if err != nil {
		_ = sender.Error(err.Error(), "agui_to_eino_failed")
		return err
	}

	ctx = context.WithValue(ctx, builtinTool.ContextKeyA2UIService, s.deps.A2UIService)
	ctx = context.WithValue(ctx, builtinTool.ContextKeyA2UISender, sender)
	ctx = context.WithValue(ctx, builtinTool.ContextKeyFileUploader, s.deps.FileUploader)
	ctx = context.WithValue(ctx, builtinTool.ContextKeySandboxBaseURL, state.sandboxBaseURL)

	if err := s.runDebugAgentModeStreaming(ctx, sender, einoMessages, runtimeConfig, state, debugDetail); err != nil {
		_ = sender.Error(err.Error(), "eino_run_failed")
		return err
	}
	return sender.Finish()
}

func (s *Service) runDebugAgentModeStreaming(ctx context.Context, sender *agui.AccumulatingSender, messages []*schema.Message, runtimeConfig *AgentRuntimeConfig, state runtimeState, detail *AgentDetailView) error {
	if len(messages) == 0 {
		return nil
	}
	compiled, err := s.buildDebugAgentModeRunner(ctx, runtimeConfig, state, detail)
	if err != nil {
		return err
	}
	if compiled.cleanup != nil {
		defer compiled.cleanup()
	}
	subAgentBridge := newSubAgentActivityBridge(sender)
	iter := compiled.runner.Run(ctx, messages)
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}
		if event.Err != nil {
			return event.Err
		}
		if len(event.RunPath) > 1 {
			if err := subAgentBridge.HandleNestedEvent(event); err != nil {
				return err
			}
			continue
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		mv := event.Output.MessageOutput
		if mv.IsStreaming {
			if err := streamMessageVariant(sender, mv, subAgentBridge); err != nil {
				return err
			}
			continue
		}
		msg, err := mv.GetMessage()
		if err != nil {
			return err
		}
		if msg == nil {
			continue
		}
		if err := agui.SendEinoMessagesAsAGUIWithHooks(sender, []*schema.Message{msg}, subAgentBridge); err != nil {
			return err
		}
	}
	return nil
}

func buildDebugAgentRequest(ctx context.Context, ownerID string, input *types.RunAgentInput) (*AgentDetailView, *AgentRuntimeConfig, error) {
	if input == nil {
		return nil, nil, fmt.Errorf("input is required")
	}
	props, err := normalizeForwardedPropsMap(input.ForwardedProps)
	if err != nil {
		return nil, nil, err
	}
	rawDefinition, ok := props[forwardedPropKeyDebugDefinition]
	if !ok {
		return nil, nil, fmt.Errorf("debug definition is required")
	}
	var definition UpsertAgentInput
	if err := decodeForwardedPropPayload(rawDefinition, &definition); err != nil {
		return nil, nil, fmt.Errorf("decode debug definition: %w", err)
	}
	detail, err := AgentDefinition.BuildDebugDetail(ctx, ownerID, definition)
	if err != nil {
		return nil, nil, err
	}

	runtimeConfig := &AgentRuntimeConfig{
		Features: map[string]any{},
	}
	runtimeConfig.Conversation.Mode = "agent"
	if rawOverride, ok := props[forwardedPropKeyDebugRuntimeOverride]; ok {
		overrideMap, err := normalizeForwardedPropsMap(rawOverride)
		if err != nil {
			return nil, nil, fmt.Errorf("normalize %s: %w", forwardedPropKeyDebugRuntimeOverride, err)
		}
		if modelID, ok := overrideMap["modelId"]; ok {
			runtimeConfig.Model.ModelID = strings.TrimSpace(fmt.Sprintf("%v", modelID))
		}
		if providerID, ok := overrideMap["providerId"]; ok {
			runtimeConfig.Model.ProviderID = strings.TrimSpace(fmt.Sprintf("%v", providerID))
		}
	}
	return detail, runtimeConfig, nil
}

func decodeForwardedPropPayload(raw any, target any) error {
	if target == nil {
		return fmt.Errorf("target is required")
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}
