package agent

import (
	"encoding/json"
	"io"
	"log"
	"openIntern/internal/services/agent/agui"
	"strings"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// streamMessageVariant 根据消息角色分发到对应的流式处理器。
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

// streamAssistantMessage 处理 assistant 流式输出，兼容 tool call 与 reasoning 片段。
func streamAssistantMessage(sender *agui.AccumulatingSender, stream *schema.StreamReader[*schema.Message]) error {
	messageID := ""
	messageStarted := false
	reasoningMessageStarted := false
	reasoningSessionStarted := false
	reasoningMessageID := ""
	reasoningSessionID := ""
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
			if !reasoningSessionStarted {
				reasoningSessionID = events.GenerateMessageID()
				if err := sender.StartReasoning(reasoningSessionID); err != nil {
					return err
				}
				reasoningSessionStarted = true
			}
			if !reasoningMessageStarted {
				reasoningMessageID = events.GenerateMessageID()
				if err := sender.StartReasoningMessage(reasoningMessageID, string(types.RoleReasoning)); err != nil {
					return err
				}
				reasoningMessageStarted = true
			}
			if err := sender.ReasoningContent(reasoningMessageID, msg.ReasoningContent); err != nil {
				return err
			}
		}
	}
	if reasoningMessageStarted {
		if err := sender.EndReasoningMessage(reasoningMessageID); err != nil {
			return err
		}
	}
	if reasoningSessionStarted {
		if err := sender.EndReasoning(reasoningSessionID); err != nil {
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

// streamToolMessage 聚合 tool 流式内容并回传单次 tool result 事件。
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

// truncatePreview 截断长文本用于日志或调试预览。
func truncatePreview(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
