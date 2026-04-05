package agent

import (
	"context"
	"fmt"
	"io"
	"log"
	"openIntern/internal/models"
	"openIntern/internal/services/agent/agui"
	builtinTool "openIntern/internal/services/builtin_tool"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/cloudwego/eino/schema"
)

// RunAgent 执行一次完整的对话回合，包括流式输出与消息落库。
func (s *Service) RunAgent(ctx context.Context, w io.Writer, input *types.RunAgentInput) error {
	if input == nil {
		return nil
	}
	state := s.snapshotState()
	ownerID := ownerIDFromContext(ctx)
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
		log.Printf("RunAgent start failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}

	historyMessages, err := s.deps.MessageStore.ListThreadMessages(ownerID, threadID)
	if err != nil {
		_ = sender.Error(err.Error(), "history_load_failed")
		log.Printf("RunAgent history load failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	mergedInput, err := mergeRunAgentInputHistory(input, historyMessages)
	if err != nil {
		_ = sender.Error(err.Error(), "history_unmarshal_failed")
		log.Printf("RunAgent history merge failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	runtimeConfig, err := applyForwardedPropsChain(ctx, mergedInput)
	if err != nil {
		_ = sender.Error(err.Error(), "forwarded_props_handle_failed")
		log.Printf("RunAgent forwarded props handle failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	preparedInput := mergedInput
	if !isAgentConversationMode(runtimeConfig) {
		preparedInput, err = injectRetrievedMemoryContext(ctx, s.deps.MemoryRetriever, mergedInput)
		if err != nil {
			log.Printf("RunAgent memory retrieval failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
			preparedInput = mergedInput
		}
	}
	compressedInput, compressionStats, err := s.compressInputContext(ctx, preparedInput, runtimeConfig, state)
	if err != nil {
		_ = sender.Error(err.Error(), "context_compress_failed")
		log.Printf("RunAgent context compression failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	if compressionStats != nil && compressionStats.Enabled && compressionStats.Triggered {
		log.Printf(
			"RunAgent context compressed thread_id=%s run_id=%s original_tokens=%d compressed_tokens=%d removed_messages=%d soft_limit=%d hard_limit=%d summary_used=%t summary_updated=%t snapshot_index=%d",
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
		log.Printf("RunAgent agui to eino failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}

	// 注入 A2UI 工具所需的 context：service 与 sender
	ctx = context.WithValue(ctx, builtinTool.ContextKeyA2UIService, s.deps.A2UIService)
	ctx = context.WithValue(ctx, builtinTool.ContextKeyA2UISender, sender)
	ctx = context.WithValue(ctx, builtinTool.ContextKeyFileUploader, s.deps.FileUploader)
	ctx = context.WithValue(ctx, builtinTool.ContextKeyUserID, ownerID)

	if isAgentConversationMode(runtimeConfig) {
		err = s.runAgentModeStreaming(ctx, sender, einoMessages, runtimeConfig, state)
	} else {
		err = s.runEinoStreaming(ctx, sender, einoMessages, runtimeConfig, state)
	}
	if err != nil {
		_ = sender.Error(err.Error(), "eino_run_failed")
		log.Printf("RunAgent eino run failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}

	if err := sender.Finish(); err != nil {
		log.Printf("RunAgent finish failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}

	flushed := acc.Flush()
	persistedMessages := make([]models.Message, 0, len(flushed)+1)
	assistantKey := buildAssistantKey(runtimeConfig)
	userMessage, err := buildUserLastMessageModel(threadID, runID, assistantKey, input.Messages)
	if err != nil {
		log.Printf("RunAgent build user message failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	if userMessage != nil {
		userMessage.UserID = ownerID
		persistedMessages = append(persistedMessages, *userMessage)
	}
	accumulatedMessages, err := buildAccumulatedMessageModels(
		threadID,
		runID,
		flushed,
		assistantKey,
	)
	if err != nil {
		log.Printf("RunAgent build persisted messages failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	for index := range accumulatedMessages {
		accumulatedMessages[index].UserID = ownerID
	}
	persistedMessages = append(persistedMessages, accumulatedMessages...)
	if err := s.deps.MessageStore.CreateMessages(persistedMessages); err != nil {
		log.Printf("RunAgent persist failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	if err := s.deps.ThreadStore.TouchThread(ownerID, threadID); err != nil {
		log.Printf("RunAgent touch thread failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	if err := scheduleThreadMemorySync(s.deps.MemorySyncStateStore, ownerID, threadID, runID); err != nil {
		log.Printf("RunAgent schedule memory sync failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
	}
	if err := ensureThreadTitle(ctx, s.deps.ThreadStore, ownerID, threadID, input.Messages, state.summaryModel); err != nil {
		log.Printf("RunAgent generate title failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
	}

	return nil
}

// scheduleThreadMemorySync marks the completed run as pending long-term memory synchronization.
func scheduleThreadMemorySync(store MemorySyncStateStore, userID, threadID, runID string) error {
	if store == nil {
		return nil
	}
	return store.ScheduleThreadSync(userID, threadID, runID)
}

// runEinoStreaming 消费 runner 事件流并按 AGUI 协议转发到 sender。
func (s *Service) runEinoStreaming(ctx context.Context, sender *agui.AccumulatingSender, messages []*schema.Message, runtimeConfig *AgentRuntimeConfig, state runtimeState) error {
	if len(messages) == 0 {
		return nil
	}
	runner, cleanup, err := s.buildEinoRunner(ctx, runtimeConfig, state)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}
	iter := runner.Run(ctx, messages)
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
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		mv := event.Output.MessageOutput
		if mv.IsStreaming {
			if err := streamMessageVariant(sender, mv, nil); err != nil {
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
		if err := agui.SendEinoMessagesAsAGUIWithHooks(sender, []*schema.Message{msg}, nil); err != nil {
			return err
		}
	}
	return nil
}

// runAgentModeStreaming compiles the selected agent tree and streams events through the shared AGUI sender.
func (s *Service) runAgentModeStreaming(ctx context.Context, sender *agui.AccumulatingSender, messages []*schema.Message, runtimeConfig *AgentRuntimeConfig, state runtimeState) error {
	if len(messages) == 0 {
		return nil
	}
	compiled, err := s.buildAgentModeRunner(ctx, runtimeConfig, state)
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
