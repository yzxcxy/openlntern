package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"openIntern/internal/models"
	"openIntern/internal/util"
	"sort"
	"strings"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
)

// mergeRunAgentInputHistory 将历史消息与本次输入合并为连续会话上下文。
func mergeRunAgentInputHistory(input *types.RunAgentInput, history []models.Message) (*types.RunAgentInput, error) {
	if input == nil {
		return nil, nil
	}
	merged := *input
	merged.Messages = nil
	for _, item := range history {
		if item.Type != "text" {
			continue
		}
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
		var lastUser *types.Message
		for i := len(input.Messages) - 1; i >= 0; i-- {
			msg := input.Messages[i]
			if msg.Role == types.RoleUser {
				lastUser = &msg
				break
			}
		}
		if lastUser != nil {
			merged.Messages = append(merged.Messages, *lastUser)
		}
	}
	return &merged, nil
}

const (
	forwardedPropKeyA2UIAction  = "a2uiAction"
	forwardedPropKeyAgentConfig = "agentConfig"
	a2uiActionPromptPrefix      = "上一条消息你发送了前端卡片，用户触发的行为以及产生的数据为："
)

type AgentRuntimeConfig struct {
	Conversation struct {
		Mode string
	}
	Model struct {
		ProviderID string
		ModelID    string
	}
	Plugins struct {
		Mode            string
		SelectedToolIDs []string
	}
	Features map[string]any
}

type forwardedPropsChainContext struct {
	props         map[string]any
	input         *types.RunAgentInput
	runtimeConfig *AgentRuntimeConfig
}

type forwardedPropsChainHandler interface {
	Handle(ctx *forwardedPropsChainContext) error
}

type a2uiActionForwardedPropsHandler struct{}
type agentConfigForwardedPropsHandler struct{}

// Handle 处理 a2uiAction，并将其转换为追加的 user 消息。
func (h a2uiActionForwardedPropsHandler) Handle(ctx *forwardedPropsChainContext) error {
	if ctx == nil || ctx.input == nil || len(ctx.props) == 0 {
		return nil
	}
	a2uiAction, ok := ctx.props[forwardedPropKeyA2UIAction]
	if !ok {
		return nil
	}
	delete(ctx.props, forwardedPropKeyA2UIAction)
	b, err := json.Marshal(a2uiAction)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", forwardedPropKeyA2UIAction, err)
	}
	ctx.input.Messages = append(ctx.input.Messages, types.Message{
		ID:      events.GenerateMessageID(),
		Role:    types.RoleUser,
		Content: a2uiActionPromptPrefix + string(b),
	})
	return nil
}

// Handle 解析 agentConfig 并写入运行时配置结构。
func (h agentConfigForwardedPropsHandler) Handle(ctx *forwardedPropsChainContext) error {
	if ctx == nil || ctx.runtimeConfig == nil || len(ctx.props) == 0 {
		return nil
	}
	raw, ok := ctx.props[forwardedPropKeyAgentConfig]
	if !ok {
		return nil
	}
	delete(ctx.props, forwardedPropKeyAgentConfig)
	cfgMap, err := normalizeForwardedPropsMap(raw)
	if err != nil {
		return fmt.Errorf("normalize %s: %w", forwardedPropKeyAgentConfig, err)
	}
	if len(cfgMap) == 0 {
		return nil
	}
	if conversationRaw, ok := cfgMap["conversation"]; ok {
		conversationMap, err := normalizeForwardedPropsMap(conversationRaw)
		if err != nil {
			return fmt.Errorf("normalize %s.conversation: %w", forwardedPropKeyAgentConfig, err)
		}
		if mode, ok := conversationMap["mode"]; ok {
			ctx.runtimeConfig.Conversation.Mode = strings.TrimSpace(fmt.Sprintf("%v", mode))
		}
	}
	if modelRaw, ok := cfgMap["model"]; ok {
		modelMap, err := normalizeForwardedPropsMap(modelRaw)
		if err != nil {
			return fmt.Errorf("normalize %s.model: %w", forwardedPropKeyAgentConfig, err)
		}
		if providerID, ok := modelMap["providerId"]; ok {
			ctx.runtimeConfig.Model.ProviderID = strings.TrimSpace(fmt.Sprintf("%v", providerID))
		}
		if modelID, ok := modelMap["modelId"]; ok {
			ctx.runtimeConfig.Model.ModelID = strings.TrimSpace(fmt.Sprintf("%v", modelID))
		}
	}
	if pluginsRaw, ok := cfgMap["plugins"]; ok {
		pluginsMap, err := normalizeForwardedPropsMap(pluginsRaw)
		if err != nil {
			return fmt.Errorf("normalize %s.plugins: %w", forwardedPropKeyAgentConfig, err)
		}
		if mode, ok := pluginsMap["mode"]; ok {
			ctx.runtimeConfig.Plugins.Mode = strings.TrimSpace(fmt.Sprintf("%v", mode))
		}
		if selectedToolIDsRaw, ok := pluginsMap["selectedToolIds"]; ok {
			selectedToolIDs, err := normalizeForwardedPropsStringList(selectedToolIDsRaw)
			if err != nil {
				return fmt.Errorf("normalize %s.plugins.selectedToolIds: %w", forwardedPropKeyAgentConfig, err)
			}
			ctx.runtimeConfig.Plugins.SelectedToolIDs = selectedToolIDs
		}
	}
	if featuresRaw, ok := cfgMap["features"]; ok {
		featuresMap, err := normalizeForwardedPropsMap(featuresRaw)
		if err != nil {
			return fmt.Errorf("normalize %s.features: %w", forwardedPropKeyAgentConfig, err)
		}
		if len(featuresMap) > 0 {
			ctx.runtimeConfig.Features = featuresMap
		}
	}
	return nil
}

// applyForwardedPropsChain 依次执行 forwarded props 处理链并返回运行时配置。
func applyForwardedPropsChain(input *types.RunAgentInput) (*AgentRuntimeConfig, error) {
	runtimeConfig := &AgentRuntimeConfig{
		Features: map[string]any{},
	}
	runtimeConfig.Conversation.Mode = "chat"
	runtimeConfig.Plugins.Mode = "select"
	if input == nil {
		return runtimeConfig, nil
	}
	props, err := normalizeForwardedPropsMap(input.ForwardedProps)
	if err != nil {
		return nil, err
	}
	if len(props) == 0 {
		return runtimeConfig, nil
	}
	chainCtx := &forwardedPropsChainContext{
		props:         props,
		input:         input,
		runtimeConfig: runtimeConfig,
	}
	handlers := []forwardedPropsChainHandler{
		a2uiActionForwardedPropsHandler{},
		agentConfigForwardedPropsHandler{},
	}
	for _, handler := range handlers {
		if err := handler.Handle(chainCtx); err != nil {
			return nil, err
		}
	}
	if len(chainCtx.props) > 0 {
		keys := make([]string, 0, len(chainCtx.props))
		for k := range chainCtx.props {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		log.Printf("RunAgent forwarded props ignored keys=%v", keys)
	}
	return chainCtx.runtimeConfig, nil
}

// normalizeForwardedPropsMap 将任意 map 输入标准化为 map[string]any。
func normalizeForwardedPropsMap(raw any) (map[string]any, error) {
	if raw == nil {
		return nil, nil
	}
	if v, ok := raw.(map[string]any); ok {
		props := make(map[string]any, len(v))
		for k, val := range v {
			props[k] = val
		}
		return props, nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	if string(b) == "null" {
		return nil, nil
	}
	var props map[string]any
	if err := json.Unmarshal(b, &props); err != nil {
		return nil, err
	}
	return props, nil
}

// normalizeForwardedPropsStringList 将字符串或字符串数组统一为去重后的字符串切片。
func normalizeForwardedPropsStringList(raw any) ([]string, error) {
	if raw == nil {
		return nil, nil
	}
	if values, ok := raw.([]string); ok {
		return util.NormalizeUniqueStringList(values), nil
	}
	if values, ok := raw.([]any); ok {
		result := make([]string, 0, len(values))
		for _, value := range values {
			text := strings.TrimSpace(fmt.Sprintf("%v", value))
			if text != "" {
				result = append(result, text)
			}
		}
		return util.NormalizeUniqueStringList(result), nil
	}
	if text, ok := raw.(string); ok {
		text = strings.TrimSpace(text)
		if text == "" {
			return nil, nil
		}
		return []string{text}, nil
	}

	b, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	if string(b) == "null" {
		return nil, nil
	}

	var values []any
	if err := json.Unmarshal(b, &values); err == nil {
		result := make([]string, 0, len(values))
		for _, value := range values {
			text := strings.TrimSpace(fmt.Sprintf("%v", value))
			if text != "" {
				result = append(result, text)
			}
		}
		return util.NormalizeUniqueStringList(result), nil
	}

	var single string
	if err := json.Unmarshal(b, &single); err == nil {
		single = strings.TrimSpace(single)
		if single == "" {
			return nil, nil
		}
		return []string{single}, nil
	}

	return nil, fmt.Errorf("must be a string or string list")
}
