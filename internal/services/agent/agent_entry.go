package agent

import (
	"context"
	"fmt"
	"io"
	"log"
	"openIntern/internal/agui"
	builtinTool "openIntern/internal/services/builtin_tool"
	"strings"

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

	historyMessages, err := s.deps.MessageStore.ListThreadMessages(threadID)
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
	runtimeConfig, err := applyForwardedPropsChain(mergedInput)
	if err != nil {
		_ = sender.Error(err.Error(), "forwarded_props_handle_failed")
		log.Printf("RunAgent forwarded props handle failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	if runtimeConfig != nil && strings.EqualFold(runtimeConfig.Conversation.Mode, "agent") {
		err := fmt.Errorf("agent mode is not available yet")
		_ = sender.Error(err.Error(), "agent_mode_not_available")
		return err
	}
	einoMessages, err := agui.AGUIRunInputToEinoMessages(mergedInput)
	if err != nil {
		_ = sender.Error(err.Error(), "agui_to_eino_failed")
		log.Printf("RunAgent agui to eino failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}

	// 注入 A2UI 工具所需的 context：service 与 sender
	ctx = context.WithValue(ctx, builtinTool.ContextKeyA2UIService, s.deps.A2UIService)
	ctx = context.WithValue(ctx, builtinTool.ContextKeyA2UISender, sender)
	ctx = context.WithValue(ctx, builtinTool.ContextKeyFileUploader, s.deps.FileUploader)
	ctx = context.WithValue(ctx, builtinTool.ContextKeySandboxBaseURL, state.sandboxBaseURL)

	err = s.runEinoStreaming(ctx, sender, einoMessages, runtimeConfig, state)
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
	if err := persistUserLastMessage(s.deps.MessageStore, threadID, runID, input.Messages); err != nil {
		log.Printf("RunAgent persist user message failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	if err := persistAccumulatedMessages(s.deps.MessageStore, threadID, runID, flushed); err != nil {
		log.Printf("RunAgent persist failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	if err := s.deps.ThreadStore.TouchThread(threadID); err != nil {
		log.Printf("RunAgent touch thread failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	if err := ensureThreadTitle(ctx, s.deps.ThreadStore, threadID, input.Messages, state.titleModel); err != nil {
		log.Printf("RunAgent generate title failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
	}

	return nil
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
			if err := streamMessageVariant(sender, mv); err != nil {
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
		if err := agui.SendEinoMessagesAsAGUI(sender, []*schema.Message{msg}); err != nil {
			return err
		}
	}
	return nil
}
