package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"openIntern/internal/agui"
	"openIntern/internal/config"
	"openIntern/internal/models"
	"openIntern/internal/services/tools"
	"strings"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/cloudwego/eino-ext/callbacks/apmplus"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func RunAgent(ctx context.Context, w io.Writer, input *types.RunAgentInput) error {
	if input == nil {
		return nil
	}
	threadID := input.ThreadID
	if threadID == "" {
		threadID = events.GenerateThreadID()
	}

	acc := agui.NewAccumulator(threadID)
	baseSender := agui.NewSenderWithThreadID(ctx, w, threadID)
	s := agui.NewAccumulatingSender(baseSender, acc)
	runID := baseSender.RunID()

	if err := s.Start(); err != nil {
		log.Printf("RunAgent start failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}

	historyMessages, err := Message.ListThreadMessages(threadID)
	if err != nil {
		_ = s.Error(err.Error(), "history_load_failed")
		log.Printf("RunAgent history load failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	mergedInput, err := mergeRunAgentInputHistory(input, historyMessages)
	if err != nil {
		_ = s.Error(err.Error(), "history_unmarshal_failed")
		log.Printf("RunAgent history merge failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	einoMessages, err := agui.AGUIRunInputToEinoMessages(mergedInput)
	if err != nil {
		_ = s.Error(err.Error(), "agui_to_eino_failed")
		log.Printf("RunAgent agui to eino failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	// 注入 A2UI 工具所需的 context：service 与 sender（user_id 由 controller 注入）
	ctx = context.WithValue(ctx, tools.ContextKeyA2UIService, A2UI)
	ctx = context.WithValue(ctx, tools.ContextKeyA2UISender, s)
	err = runEinoStreaming(ctx, s, einoMessages)
	if err != nil {
		_ = s.Error(err.Error(), "eino_run_failed")
		log.Printf("RunAgent eino run failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}

	if err := s.Finish(); err != nil {
		log.Printf("RunAgent finish failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}

	flushed := acc.Flush()
	if err := persistUserLastMessage(threadID, input.Messages); err != nil {
		log.Printf("RunAgent persist user message failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	if err := persistAccumulatedMessages(threadID, flushed); err != nil {
		log.Printf("RunAgent persist failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	if err := Thread.TouchThread(threadID); err != nil {
		log.Printf("RunAgent touch thread failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}

	return nil
}

var einoRunner *adk.Runner
var apmplusShutdown func(context.Context) error

func InitEino(cfg config.LLMConfig, toolsCfg config.ToolsConfig, apmCfg config.APMPlusConfig) (func(context.Context) error, error) {
	ctx := context.Background()
	if apmCfg.Host != "" && apmCfg.AppKey != "" && apmCfg.ServiceName != "" {
		cbh, shutdown, err := apmplus.NewApmplusHandler(&apmplus.Config{
			Host:        apmCfg.Host,
			AppKey:      apmCfg.AppKey,
			ServiceName: apmCfg.ServiceName,
			Release:     apmCfg.Release,
		})
		if err != nil {
			return nil, err
		}
		callbacks.AppendGlobalHandlers(cbh)
		apmplusShutdown = shutdown
	}
	if apmplusShutdown == nil {
		apmplusShutdown = func(context.Context) error { return nil }
	}
	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey: cfg.APIKey,
		Model:  cfg.Model,
	})
	if err != nil {
		return nil, err
	}
	sandboxTools, err := tools.GetSandboxTools(ctx, toolsCfg.Sandbox.Url)
	if err != nil {
		return nil, err
	}
	a2uiTools, err := tools.GetA2UITools(ctx)
	if err != nil {
		return nil, err
	}
	allTools := append(sandboxTools, a2uiTools...)
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "openintern_agent",
		Description: "openintern agent",
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: allTools,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	einoRunner = adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent, EnableStreaming: true})
	return apmplusShutdown, nil
}

func mergeRunAgentInputHistory(input *types.RunAgentInput, history []models.Message) (*types.RunAgentInput, error) {
	if input == nil {
		return nil, nil
	}
	merged := *input
	merged.Messages = nil
	for _, item := range history {
		if item.Content == "" {
			continue
		}
		var raw map[string]any
		if err := json.Unmarshal([]byte(item.Content), &raw); err == nil {
			if _, ok := raw["messages"]; ok {
				var past types.RunAgentInput
				if err := json.Unmarshal([]byte(item.Content), &past); err != nil {
					return nil, err
				}
				if len(past.Messages) > 0 {
					merged.Messages = append(merged.Messages, past.Messages...)
				}
				continue
			}
			var pastMsg types.Message
			if err := json.Unmarshal([]byte(item.Content), &pastMsg); err != nil {
				return nil, err
			}
			if pastMsg.ID == "" {
				pastMsg.ID = item.MsgID
			}
			merged.Messages = append(merged.Messages, pastMsg)
			continue
		}
		role := ""
		if item.Metadata != "" {
			var meta map[string]any
			if err := json.Unmarshal([]byte(item.Metadata), &meta); err != nil {
				return nil, err
			}
			if v, ok := meta["role"].(string); ok {
				role = v
			}
		}
		if role == "" {
			role = string(types.RoleAssistant)
		}
		merged.Messages = append(merged.Messages, types.Message{
			ID:      item.MsgID,
			Role:    types.Role(role),
			Content: item.Content,
		})
	}
	if len(input.Messages) > 0 {
		merged.Messages = append(merged.Messages, input.Messages...)
	}
	return &merged, nil
}

func runEinoStreaming(ctx context.Context, sender *agui.AccumulatingSender, messages []*schema.Message) error {
	if len(messages) == 0 {
		return nil
	}
	if einoRunner == nil {
		return fmt.Errorf("eino runner not initialized")
	}
	iter := einoRunner.Run(ctx, messages)
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

func streamMessageVariant(sender *agui.AccumulatingSender, mv *adk.MessageVariant) error {
	if mv == nil || mv.MessageStream == nil {
		return nil
	}
	defer mv.MessageStream.Close()
	switch mv.Role {
	case schema.Assistant:
		return streamAssistantMessage(sender, mv.MessageStream)
	case schema.Tool:
		return streamToolMessage(sender, mv.MessageStream)
	default:
		for {
			_, err := mv.MessageStream.Recv()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
		}
	}
}

func streamAssistantMessage(sender *agui.AccumulatingSender, stream *schema.StreamReader[*schema.Message]) error {
	messageID := ""
	messageStarted := false
	thinkingStarted := false
	thinkingSessionStarted := false
	toolCallStarted := map[string]bool{}
	toolCallArgs := map[string]string{}
	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if msg == nil {
			continue
		}
		if len(msg.ToolCalls) > 0 {
			for _, call := range msg.ToolCalls {
				log.Printf("tool call: %v", call)
				callID := call.ID
				// 流式可能在同一批里先发带 id+name 的项，再发无 id、无 name 仅带 args 的项（解析产物），
				// 需在「处理当前项时」再算 onlyStartedID，这样同一批里后一项才能归到已开始的那一个。
				if callID == "" && call.Function.Name == "" {
					var onlyStartedID string
					if len(toolCallStarted) == 1 {
						for id := range toolCallStarted {
							onlyStartedID = id
							break
						}
					}
					if onlyStartedID != "" {
						callID = onlyStartedID
					}
				}
				if callID == "" {
					callID = events.GenerateToolCallID()
				}
				// 流式解析可能先发出带 name 的 chunk，再发出仅带 partial args 的 chunk（name 为空），
				// 若对 name 为空的 chunk 调用 StartToolCall 会触发 "toolCallName field is required" 校验失败。
				if call.Function.Name != "" && !toolCallStarted[callID] {
					if err := sender.StartToolCall(callID, call.Function.Name); err != nil {
						return err
					}
					toolCallStarted[callID] = true
				}
				args := call.Function.Arguments
				// 仅对已发起过的 tool call 追加 args，避免流式里仅带 partial args 且无 name 的 chunk 产生未开始的 call
				if args != "" && toolCallStarted[callID] {
					prev := toolCallArgs[callID]
					delta := args
					if prev != "" && strings.HasPrefix(args, prev) {
						delta = args[len(prev):]
					}
					if delta != "" {
						if err := sender.ToolCallArgs(callID, delta); err != nil {
							return err
						}
					}
					toolCallArgs[callID] = args
				}
			}
		}
		if msg.Content != "" {
			if !messageStarted {
				messageID = events.GenerateMessageID()
				if err := sender.StartMessage(messageID, string(schema.Assistant)); err != nil {
					return err
				}
				messageStarted = true
			}
			if err := sender.SendContent(messageID, msg.Content); err != nil {
				return err
			}
		}
		if len(msg.AssistantGenMultiContent) > 0 {
			if !messageStarted {
				messageID = events.GenerateMessageID()
				if err := sender.StartMessage(messageID, string(schema.Assistant)); err != nil {
					return err
				}
				messageStarted = true
			}
			if err := sender.Custom("agui-llm:assistant_multimodal", map[string]any{
				"msg_id": messageID,
				"parts":  msg.AssistantGenMultiContent,
			}); err != nil {
				return err
			}
		}
		if msg.ReasoningContent != "" {
			if !thinkingSessionStarted {
				if err := sender.StartThinking(""); err != nil {
					return err
				}
				thinkingSessionStarted = true
			}
			if !thinkingStarted {
				if err := sender.StartThinkingMessage(); err != nil {
					return err
				}
				thinkingStarted = true
			}
			if err := sender.ThinkingContent(msg.ReasoningContent); err != nil {
				return err
			}
		}
	}
	if thinkingStarted {
		if err := sender.EndThinkingMessage(); err != nil {
			return err
		}
	}
	if thinkingSessionStarted {
		if err := sender.EndThinking(); err != nil {
			return err
		}
	}
	for callID := range toolCallStarted {
		args := toolCallArgs[callID]
		if args != "" && isLikelyTruncatedJSON(args) {
			log.Printf("tool call args likely truncated (invalid JSON) call_id=%s args=%q", callID, args)
			_ = sender.Custom("agui:tool_call_args_truncated", map[string]any{
				"tool_call_id": callID,
				"raw_preview":  truncatePreview(args, 200),
			})
		}
		if err := sender.EndToolCall(callID); err != nil {
			return err
		}
	}
	if messageStarted {
		if err := sender.EndMessage(messageID); err != nil {
			return err
		}
	}
	return nil
}

func streamToolMessage(sender *agui.AccumulatingSender, stream *schema.StreamReader[*schema.Message]) error {
	var contentBuilder strings.Builder
	toolCallID := ""
	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if msg == nil {
			continue
		}
		if msg.ToolCallID != "" && toolCallID == "" {
			toolCallID = msg.ToolCallID
		}
		if msg.Content != "" {
			contentBuilder.WriteString(msg.Content)
		}
	}
	if toolCallID == "" {
		toolCallID = events.GenerateToolCallID()
	}
	return sender.ToolCallResult(events.GenerateMessageID(), toolCallID, contentBuilder.String())
}

func persistUserLastMessage(threadID string, messages []types.Message) error {
	if len(messages) == 0 {
		return nil
	}
	var lastUser *types.Message
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role == types.RoleUser {
			lastUser = &msg
			break
		}
	}
	if lastUser == nil {
		return nil
	}
	msgID := lastUser.ID
	b, err := json.Marshal(lastUser)
	if err != nil {
		return err
	}
	return Message.CreateMessages([]models.Message{
		{
			MsgID:    msgID,
			ThreadID: threadID,
			Type:     "text",
			Content:  string(b),
			Status:   "completed",
		},
	})
}

func persistAccumulatedMessages(threadID string, messages []agui.AccumulatedMessage) error {
	if len(messages) == 0 {
		return nil
	}
	modelsMessages := make([]models.Message, 0, len(messages))
	for _, msg := range messages {
		content, metadata := buildMessageContentAndMetadata(msg)
		if content == "" {
			continue
		}
		modelsMessages = append(modelsMessages, models.Message{
			MsgID:    msg.MsgID,
			ThreadID: threadID,
			Type:     msg.Type,
			Content:  content,
			Status:   "completed",
			Metadata: metadata,
		})
	}
	return Message.CreateMessages(modelsMessages)
}

func buildMessageContentAndMetadata(msg agui.AccumulatedMessage) (string, string) {
	message := types.Message{ID: msg.MsgID}
	switch msg.Type {
	case "text":
		message.Role = types.Role(msg.Role)
		message.Content = msg.Content
	case "thinking", "thinking_message":
		message.Role = types.Role("reasoning")
		message.Content = msg.Content
	case "tool_call":
		message.Role = types.RoleAssistant
		toolCallID := msg.ToolCallID
		if toolCallID == "" {
			toolCallID = msg.MsgID
		}
		args := ""
		if v, ok := msg.Content.(string); ok {
			args = v
		} else if msg.Content != nil {
			if b, err := json.Marshal(msg.Content); err == nil {
				args = string(b)
			} else {
				args = fmt.Sprintf("%v", msg.Content)
			}
		}
		toolCall := types.ToolCall{
			ID:   toolCallID,
			Type: types.ToolCallTypeFunction,
			Function: types.FunctionCall{
				Name:      msg.ToolName,
				Arguments: args,
			},
		}
		message.ToolCalls = []types.ToolCall{toolCall}
	case "tool_result":
		message.Role = types.RoleTool
		message.ToolCallID = msg.ToolCallID
		if message.ToolCallID == "" {
			message.ToolCallID = msg.MsgID
		}
		message.Content = msg.Content
	case "activity":
		message.Role = types.RoleActivity
		message.ActivityType = msg.ActivityType
		message.Content = msg.Content
	case "state", "custom", "raw", "error":
		message.Role = types.RoleActivity
		message.ActivityType = msg.Type
		message.Content = msg.Content
	default:
		if msg.Role != "" {
			message.Role = types.Role(msg.Role)
		} else {
			message.Role = types.RoleActivity
			message.ActivityType = msg.Type
		}
		message.Content = msg.Content
	}
	b, err := json.Marshal(message)
	if err != nil {
		return "", ""
	}
	content := string(b)
	meta := map[string]any{}
	for k, v := range msg.Metadata {
		meta[k] = v
	}
	metadata := ""
	if len(meta) > 0 {
		if b, err := json.Marshal(meta); err == nil {
			metadata = string(b)
		}
	}
	return content, metadata
}

// isLikelyTruncatedJSON 判断字符串是否像被截断的 JSON（非空、形似 JSON 但解析失败）。
func isLikelyTruncatedJSON(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	// 工具参数一般为 JSON 对象
	if !strings.HasPrefix(s, "{") {
		return false
	}
	return !json.Valid([]byte(s))
}

func truncatePreview(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
