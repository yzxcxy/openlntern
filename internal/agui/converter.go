package agui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

func AGUIRunInputToEinoMessages(input *types.RunAgentInput) ([]*schema.Message, error) {
	if input == nil || len(input.Messages) == 0 {
		return nil, nil
	}

	b, err := json.Marshal(input.Messages)
	if err != nil {
		return nil, err
	}
	var raw []map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	result := make([]*schema.Message, 0, len(raw))
	for _, item := range raw {
		msg := &schema.Message{}
		if role, ok := item["role"].(string); ok {
			msg.Role = schema.RoleType(role)
		}
		if name, ok := item["name"].(string); ok {
			msg.Name = name
		}
		if reasoning, ok := item["reasoning_content"].(string); ok {
			msg.ReasoningContent = reasoning
		}
		if content, ok := item["content"]; ok {
			applyContent(msg, content)
		}
		if toolCalls, ok := item["tool_calls"]; ok {
			msg.ToolCalls = parseToolCalls(toolCalls)
		}
		if toolCallID, ok := item["tool_call_id"].(string); ok {
			msg.ToolCallID = toolCallID
		}
		if toolName, ok := item["tool_name"].(string); ok {
			msg.ToolName = toolName
		}
		if extra, ok := item["extra"].(map[string]any); ok {
			msg.Extra = extra
		}
		result = append(result, msg)
	}
	return result, nil
}

func SendEinoMessagesAsAGUI(sender *AccumulatingSender, messages []*schema.Message) error {
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		switch msg.Role {
		case schema.Assistant:
			if len(msg.ToolCalls) > 0 {
				for _, call := range msg.ToolCalls {
					if err := sender.StartToolCall(call.ID, call.Function.Name); err != nil {
						return err
					}
					if call.Function.Arguments != "" {
						if err := sender.ToolCallArgs(call.ID, call.Function.Arguments); err != nil {
							return err
						}
					}
					if err := sender.EndToolCall(call.ID); err != nil {
						return err
					}
				}
			}
			if msg.Content != "" || len(msg.AssistantGenMultiContent) > 0 {
				msgID := uuid.NewString()
				if err := sender.StartMessage(msgID, string(schema.Assistant)); err != nil {
					return err
				}
				text := msg.Content
				if text == "" {
					text = concatOutputText(msg.AssistantGenMultiContent)
				}
				if text != "" {
					if err := sender.SendContent(msgID, text); err != nil {
						return err
					}
				}
				if len(msg.AssistantGenMultiContent) > 0 {
					if err := sender.Custom("agui-llm:assistant_multimodal", map[string]any{
						"msg_id": msgID,
						"parts":  msg.AssistantGenMultiContent,
					}); err != nil {
						return err
					}
				}
				if err := sender.EndMessage(msgID); err != nil {
					return err
				}
			}
		case schema.Tool:
			msgID := uuid.NewString()
			toolCallID := msg.ToolCallID
			if toolCallID == "" {
				toolCallID = msgID
			}
			content := msg.Content
			if content == "" {
				content = concatInputText(msg.UserInputMultiContent)
			}
			if err := sender.ToolCallResult(msgID, toolCallID, content); err != nil {
				return err
			}
		}
	}
	return nil
}

func applyContent(msg *schema.Message, content any) {
	switch v := content.(type) {
	case string:
		msg.Content = v
	case []any:
		if msg.Role == schema.User {
			msg.UserInputMultiContent = parseInputParts(v)
		} else if msg.Role == schema.Assistant {
			msg.AssistantGenMultiContent = parseOutputParts(v)
		} else {
			msg.Content = stringify(content)
		}
	case map[string]any:
		if text, ok := v["text"].(string); ok && text != "" {
			msg.Content = text
			return
		}
		msg.Content = stringify(content)
	default:
		msg.Content = stringify(content)
	}
}

func parseToolCalls(value any) []schema.ToolCall {
	rawList, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]schema.ToolCall, 0, len(rawList))
	for _, item := range rawList {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		call := schema.ToolCall{}
		if id, ok := entry["id"].(string); ok {
			call.ID = id
		}
		if typ, ok := entry["type"].(string); ok {
			call.Type = typ
		}
		if idx, ok := entry["index"].(float64); ok {
			i := int(idx)
			call.Index = &i
		}
		if fn, ok := entry["function"].(map[string]any); ok {
			if name, ok := fn["name"].(string); ok {
				call.Function.Name = name
			}
			if args, ok := fn["arguments"].(string); ok {
				call.Function.Arguments = args
			}
		}
		if len(call.Extra) == 0 {
			if extra, ok := entry["extra"].(map[string]any); ok {
				call.Extra = extra
			}
		}
		result = append(result, call)
	}
	return result
}

func parseInputParts(values []any) []schema.MessageInputPart {
	parts := make([]schema.MessageInputPart, 0, len(values))
	for _, item := range values {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		typ, _ := entry["type"].(string)
		part := schema.MessageInputPart{Type: schema.ChatMessagePartType(typ)}
		switch typ {
		case "text":
			part.Text, _ = entry["text"].(string)
		case "image_url":
			part.Image = parseInputImage(entry["image"])
		case "audio_url":
			part.Audio = parseInputAudio(entry["audio"])
		case "video_url":
			part.Video = parseInputVideo(entry["video"])
		case "file_url":
			part.File = parseInputFile(entry["file"])
		}
		if extra, ok := entry["extra"].(map[string]any); ok {
			part.Extra = extra
		}
		parts = append(parts, part)
	}
	return parts
}

func parseOutputParts(values []any) []schema.MessageOutputPart {
	parts := make([]schema.MessageOutputPart, 0, len(values))
	for _, item := range values {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		typ, _ := entry["type"].(string)
		part := schema.MessageOutputPart{Type: schema.ChatMessagePartType(typ)}
		switch typ {
		case "text":
			part.Text, _ = entry["text"].(string)
		case "image_url":
			part.Image = parseOutputImage(entry["image"])
		case "audio_url":
			part.Audio = parseOutputAudio(entry["audio"])
		case "video_url":
			part.Video = parseOutputVideo(entry["video"])
		}
		if extra, ok := entry["extra"].(map[string]any); ok {
			part.Extra = extra
		}
		parts = append(parts, part)
	}
	return parts
}

func parseInputImage(v any) *schema.MessageInputImage {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	img := &schema.MessageInputImage{}
	img.MessagePartCommon = parseCommon(m)
	if detail, ok := m["detail"].(string); ok {
		img.Detail = schema.ImageURLDetail(detail)
	}
	return img
}

func parseInputAudio(v any) *schema.MessageInputAudio {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	audio := &schema.MessageInputAudio{}
	audio.MessagePartCommon = parseCommon(m)
	return audio
}

func parseInputVideo(v any) *schema.MessageInputVideo {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	video := &schema.MessageInputVideo{}
	video.MessagePartCommon = parseCommon(m)
	return video
}

func parseInputFile(v any) *schema.MessageInputFile {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	file := &schema.MessageInputFile{}
	file.MessagePartCommon = parseCommon(m)
	if name, ok := m["name"].(string); ok {
		file.Name = name
	}
	return file
}

func parseOutputImage(v any) *schema.MessageOutputImage {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	img := &schema.MessageOutputImage{}
	img.MessagePartCommon = parseCommon(m)
	return img
}

func parseOutputAudio(v any) *schema.MessageOutputAudio {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	audio := &schema.MessageOutputAudio{}
	audio.MessagePartCommon = parseCommon(m)
	return audio
}

func parseOutputVideo(v any) *schema.MessageOutputVideo {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	video := &schema.MessageOutputVideo{}
	video.MessagePartCommon = parseCommon(m)
	return video
}

func parseCommon(m map[string]any) schema.MessagePartCommon {
	common := schema.MessagePartCommon{}
	if url, ok := m["url"].(string); ok && url != "" {
		common.URL = &url
	}
	if base64data, ok := m["base64data"].(string); ok && base64data != "" {
		common.Base64Data = &base64data
	}
	if mime, ok := m["mime_type"].(string); ok {
		common.MIMEType = mime
	}
	if extra, ok := m["extra"].(map[string]any); ok {
		common.Extra = extra
	}
	return common
}

func concatOutputText(parts []schema.MessageOutputPart) string {
	var sb strings.Builder
	for _, part := range parts {
		if part.Type == schema.ChatMessagePartTypeText && part.Text != "" {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(part.Text)
		}
	}
	return sb.String()
}

func concatInputText(parts []schema.MessageInputPart) string {
	var sb strings.Builder
	for _, part := range parts {
		if part.Type == schema.ChatMessagePartTypeText && part.Text != "" {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(part.Text)
		}
	}
	return sb.String()
}

func stringify(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}
