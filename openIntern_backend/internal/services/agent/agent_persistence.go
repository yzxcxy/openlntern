package agent

import (
	"encoding/json"
	"fmt"
	"openIntern/internal/models"
	"openIntern/internal/services/agent/agui"
	"strings"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/google/uuid"
)

const (
	chatAssistantKey        = "chat"
	agentAssistantKeyPrefix = "agent:"
)

// buildUserLastMessageModel 构建本次输入中的最后一条 user 消息模型。
func buildUserLastMessageModel(threadID, runID, assistantKey string, messages []types.Message) (*models.Message, error) {
	if len(messages) == 0 {
		return nil, nil
	}
	if runID == "" {
		return nil, fmt.Errorf("run_id is required")
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
		return nil, nil
	}
	msgID := normalizeMsgID(lastUser.ID)
	normalized := *lastUser
	normalized.ID = msgID
	b, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	metadata := ""
	if strings.TrimSpace(assistantKey) != "" {
		meta := map[string]any{
			"assistant_key": strings.TrimSpace(assistantKey),
		}
		if metadataBytes, marshalErr := json.Marshal(meta); marshalErr == nil {
			metadata = string(metadataBytes)
		}
	}
	return &models.Message{
		MsgID:    msgID,
		ThreadID: threadID,
		RunID:    runID,
		Type:     "text",
		Content:  string(b),
		Status:   "completed",
		Metadata: metadata,
	}, nil
}

// buildAccumulatedMessageModels 构建流式累计消息的持久化模型集合。
func buildAccumulatedMessageModels(threadID, runID string, messages []agui.AccumulatedMessage, assistantKey string) ([]models.Message, error) {
	if len(messages) == 0 {
		return nil, nil
	}
	if runID == "" {
		return nil, fmt.Errorf("run_id is required")
	}
	modelsMessages := make([]models.Message, 0, len(messages))
	for _, msg := range messages {
		msg.MsgID = normalizeMsgID(msg.MsgID)
		content, metadata := buildMessageContentAndMetadata(msg, assistantKey)
		if content == "" {
			continue
		}
		modelsMessages = append(modelsMessages, models.Message{
			MsgID:    msg.MsgID,
			ThreadID: threadID,
			RunID:    runID,
			Type:     msg.Type,
			Content:  content,
			Status:   "completed",
			Metadata: metadata,
		})
	}
	return modelsMessages, nil
}

// normalizeMsgID 标准化消息 ID，必要时生成新 UUID。
func normalizeMsgID(input string) string {
	id := strings.TrimSpace(input)
	id = strings.TrimPrefix(id, "msg-")
	if id == "" {
		return uuid.NewString()
	}
	if _, err := uuid.Parse(id); err == nil {
		return id
	}
	if len(id) > 36 {
		return uuid.NewString()
	}
	return id
}

// buildMessageContentAndMetadata 将累计消息转换为存储层 content/metadata 格式。
func buildMessageContentAndMetadata(msg agui.AccumulatedMessage, assistantKey string) (string, string) {
	message := types.Message{ID: msg.MsgID}
	switch msg.Type {
	case "text":
		message.Role = types.Role(msg.Role)
		message.Content = msg.Content
	case "reasoning_message":
		message.Role = types.RoleReasoning
		message.Content = msg.Content
	case "tool_call":
		// 注意tool_call是内嵌在assistant消息中的，和tool_result是不同的角色
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
	// assistant_key 只保存稳定身份，避免把名称和头像冗余落库。
	if strings.TrimSpace(assistantKey) != "" {
		meta["assistant_key"] = strings.TrimSpace(assistantKey)
	}
	metadata := ""
	if len(meta) > 0 {
		if b, err := json.Marshal(meta); err == nil {
			metadata = string(b)
		}
	}
	return content, metadata
}

// buildAssistantKey 为一次 run 生成稳定的 assistant 身份标识。
func buildAssistantKey(runtimeConfig *AgentRuntimeConfig) string {
	if !isAgentConversationMode(runtimeConfig) {
		return chatAssistantKey
	}
	selectedAgentID := selectedAgentIDFromRuntimeConfig(runtimeConfig)
	if selectedAgentID == "" {
		return chatAssistantKey
	}
	return agentAssistantKeyPrefix + selectedAgentID
}
