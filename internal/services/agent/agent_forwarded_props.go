package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"openIntern/internal/dao"
	"openIntern/internal/models"
	"openIntern/internal/util"
	"sort"
	"strconv"
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
	forwardedPropKeyA2UIAction        = "a2uiAction"
	forwardedPropKeyAgentConfig       = "agentConfig"
	forwardedPropKeyContextSelections = "contextSelections"
	a2uiActionPromptPrefix            = "上一条消息你发送了前端卡片，用户触发的行为以及产生的数据为："
)

const (
	knowledgeSearchLimitPerKB = 3
)

const (
	defaultPluginSearchTopK        = 10
	defaultPluginSearchMaxMCPTools = 3
)

const (
	skillInstructionPrefix = "请优先参考并执行以下技能约束："
	kbContextPrefix        = "以下是当前问题在所选知识库中的检索结果，请优先参考这些信息后再回答："
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
		Search          struct {
			TopK         int
			RuntimeTypes []string
			MinScore     float64
			MaxMCPTools  int
		}
		SearchQuery string
	}
	Features map[string]any
}

type forwardedPropsChainContext struct {
	requestCtx    context.Context
	props         map[string]any
	input         *types.RunAgentInput
	runtimeConfig *AgentRuntimeConfig
}

type forwardedPropsChainHandler interface {
	Handle(ctx *forwardedPropsChainContext) error
}

type a2uiActionForwardedPropsHandler struct{}
type agentConfigForwardedPropsHandler struct{}
type contextSelectionsForwardedPropsHandler struct{}

type forwardedContextSelection struct {
	Skills         []forwardedContextTarget
	KnowledgeBases []forwardedContextTarget
}

type forwardedContextTarget struct {
	ID   string
	Name string
}

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
		if searchRaw, ok := pluginsMap["search"]; ok {
			searchMap, err := normalizeForwardedPropsMap(searchRaw)
			if err != nil {
				return fmt.Errorf("normalize %s.plugins.search: %w", forwardedPropKeyAgentConfig, err)
			}
			if topKRaw, ok := searchMap["topK"]; ok {
				topK, err := normalizeForwardedPropsInt(topKRaw)
				if err != nil {
					return fmt.Errorf("normalize %s.plugins.search.topK: %w", forwardedPropKeyAgentConfig, err)
				}
				ctx.runtimeConfig.Plugins.Search.TopK = topK
			}
			if runtimeTypesRaw, ok := searchMap["runtimeTypes"]; ok {
				runtimeTypes, err := normalizeForwardedPropsStringList(runtimeTypesRaw)
				if err != nil {
					return fmt.Errorf("normalize %s.plugins.search.runtimeTypes: %w", forwardedPropKeyAgentConfig, err)
				}
				ctx.runtimeConfig.Plugins.Search.RuntimeTypes = runtimeTypes
			}
			if minScoreRaw, ok := searchMap["minScore"]; ok {
				minScore, err := normalizeForwardedPropsFloat(minScoreRaw)
				if err != nil {
					return fmt.Errorf("normalize %s.plugins.search.minScore: %w", forwardedPropKeyAgentConfig, err)
				}
				ctx.runtimeConfig.Plugins.Search.MinScore = minScore
			}
			if maxMCPToolsRaw, ok := searchMap["maxMCPTools"]; ok {
				maxMCPTools, err := normalizeForwardedPropsInt(maxMCPToolsRaw)
				if err != nil {
					return fmt.Errorf("normalize %s.plugins.search.maxMCPTools: %w", forwardedPropKeyAgentConfig, err)
				}
				ctx.runtimeConfig.Plugins.Search.MaxMCPTools = maxMCPTools
			}
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

// Handle 处理 contextSelections：检索知识库并在用户消息前插入临时上下文，同时插入技能约束临时消息。
func (h contextSelectionsForwardedPropsHandler) Handle(ctx *forwardedPropsChainContext) error {
	if ctx == nil || ctx.input == nil || len(ctx.props) == 0 {
		return nil
	}
	raw, ok := ctx.props[forwardedPropKeyContextSelections]
	if !ok {
		return nil
	}
	delete(ctx.props, forwardedPropKeyContextSelections)
	selection, err := normalizeForwardedContextSelection(raw)
	if err != nil {
		return fmt.Errorf("normalize %s: %w", forwardedPropKeyContextSelections, err)
	}
	if selection == nil {
		return nil
	}

	userText, userIndex := findLastUserMessageTextAndIndex(ctx.input.Messages)
	if len(selection.KnowledgeBases) > 0 && userText != "" {
		matches, err := dao.KnowledgeBase.SearchInKnowledgeBases(
			ctx.requestCtx,
			userText,
			collectKnowledgeBaseNames(selection.KnowledgeBases),
			knowledgeSearchLimitPerKB,
		)
		if err != nil {
			return fmt.Errorf("search knowledge bases: %w", err)
		}
		if len(matches) > 0 {
			injectMessageBeforeUserAt(ctx.input, userIndex, types.Message{
				ID:      events.GenerateMessageID(),
				Role:    types.RoleSystem,
				Content: buildKnowledgeBaseContextMessage(matches),
			})
			_, userIndex = findLastUserMessageTextAndIndex(ctx.input.Messages)
		}
	}
	if len(selection.Skills) > 0 {
		injectMessageBeforeUserAt(ctx.input, userIndex, types.Message{
			ID:      events.GenerateMessageID(),
			Role:    types.RoleUser,
			Content: buildSkillInstructionMessage(selection.Skills),
		})
	}
	return nil
}

// applyForwardedPropsChain 依次执行 forwarded props 处理链并返回运行时配置。
func applyForwardedPropsChain(requestCtx context.Context, input *types.RunAgentInput) (*AgentRuntimeConfig, error) {
	runtimeConfig := &AgentRuntimeConfig{
		Features: map[string]any{},
	}
	runtimeConfig.Conversation.Mode = "chat"
	runtimeConfig.Plugins.Mode = "select"
	runtimeConfig.Plugins.Search.TopK = defaultPluginSearchTopK
	runtimeConfig.Plugins.Search.MaxMCPTools = defaultPluginSearchMaxMCPTools
	if requestCtx == nil {
		requestCtx = context.Background()
	}
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
		requestCtx:    requestCtx,
		props:         props,
		input:         input,
		runtimeConfig: runtimeConfig,
	}
	handlers := []forwardedPropsChainHandler{
		a2uiActionForwardedPropsHandler{},
		agentConfigForwardedPropsHandler{},
		contextSelectionsForwardedPropsHandler{},
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
	runtimeConfig.Plugins.Search.RuntimeTypes = normalizePluginSearchRuntimeTypes(runtimeConfig.Plugins.Search.RuntimeTypes)
	if strings.EqualFold(runtimeConfig.Plugins.Mode, "search") {
		runtimeConfig.Plugins.SearchQuery, _ = findLastUserMessageTextAndIndex(input.Messages)
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

// normalizeForwardedPropsInt 将任意整型输入标准化为 int。
func normalizeForwardedPropsInt(raw any) (int, error) {
	if raw == nil {
		return 0, nil
	}
	switch value := raw.(type) {
	case int:
		return value, nil
	case int8:
		return int(value), nil
	case int16:
		return int(value), nil
	case int32:
		return int(value), nil
	case int64:
		return int(value), nil
	case float32:
		return int(value), nil
	case float64:
		return int(value), nil
	case json.Number:
		n, err := value.Int64()
		if err != nil {
			return 0, err
		}
		return int(n), nil
	case string:
		text := strings.TrimSpace(value)
		if text == "" {
			return 0, nil
		}
		n, err := strconv.Atoi(text)
		if err != nil {
			return 0, err
		}
		return n, nil
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if text == "" || text == "<nil>" {
			return 0, nil
		}
		n, err := strconv.Atoi(text)
		if err != nil {
			return 0, err
		}
		return n, nil
	}
}

// normalizeForwardedPropsFloat 将任意浮点输入标准化为 float64。
func normalizeForwardedPropsFloat(raw any) (float64, error) {
	if raw == nil {
		return 0, nil
	}
	switch value := raw.(type) {
	case float64:
		return value, nil
	case float32:
		return float64(value), nil
	case int:
		return float64(value), nil
	case int8:
		return float64(value), nil
	case int16:
		return float64(value), nil
	case int32:
		return float64(value), nil
	case int64:
		return float64(value), nil
	case json.Number:
		n, err := value.Float64()
		if err != nil {
			return 0, err
		}
		return n, nil
	case string:
		text := strings.TrimSpace(value)
		if text == "" {
			return 0, nil
		}
		n, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return 0, err
		}
		return n, nil
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if text == "" || text == "<nil>" {
			return 0, nil
		}
		n, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return 0, err
		}
		return n, nil
	}
}

// normalizePluginSearchRuntimeTypes 仅保留支持的 runtime_type 并去重。
func normalizePluginSearchRuntimeTypes(runtimeTypes []string) []string {
	normalized := make([]string, 0, len(runtimeTypes))
	for _, item := range runtimeTypes {
		switch strings.ToLower(strings.TrimSpace(item)) {
		case "api", "mcp", "code":
			normalized = append(normalized, strings.ToLower(strings.TrimSpace(item)))
		}
	}
	return util.NormalizeUniqueStringList(normalized)
}

// normalizeForwardedContextSelection 将 forwarded contextSelections 结构标准化为内部结构。
func normalizeForwardedContextSelection(raw any) (*forwardedContextSelection, error) {
	mapped, err := normalizeForwardedPropsMap(raw)
	if err != nil {
		return nil, err
	}
	if len(mapped) == 0 {
		return nil, nil
	}
	skills, err := normalizeForwardedContextTargets(mapped["skills"])
	if err != nil {
		return nil, fmt.Errorf("skills: %w", err)
	}
	kbs, err := normalizeForwardedContextTargets(mapped["knowledgeBases"])
	if err != nil {
		return nil, fmt.Errorf("knowledgeBases: %w", err)
	}
	return &forwardedContextSelection{
		Skills:         skills,
		KnowledgeBases: kbs,
	}, nil
}

// normalizeForwardedContextTargets 标准化 skill/kb 目标列表，支持字符串或对象数组。
func normalizeForwardedContextTargets(raw any) ([]forwardedContextTarget, error) {
	if raw == nil {
		return nil, nil
	}
	switch raw.(type) {
	case string, []string:
		values, err := normalizeForwardedPropsStringList(raw)
		if err != nil {
			return nil, err
		}
		result := make([]forwardedContextTarget, 0, len(values))
		for _, item := range values {
			result = append(result, forwardedContextTarget{
				ID:   item,
				Name: item,
			})
		}
		return result, nil
	}
	items, ok := raw.([]any)
	if !ok {
		b, marshalErr := json.Marshal(raw)
		if marshalErr != nil {
			return nil, marshalErr
		}
		if string(b) == "null" {
			return nil, nil
		}
		if unmarshalErr := json.Unmarshal(b, &items); unmarshalErr != nil {
			return nil, fmt.Errorf("must be a string list or object list")
		}
	}
	result := make([]forwardedContextTarget, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		entryMap, mapErr := normalizeForwardedPropsMap(item)
		if mapErr != nil || len(entryMap) == 0 {
			text := normalizeForwardedString(item)
			if text == "" {
				continue
			}
			key := text + "|" + text
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, forwardedContextTarget{ID: text, Name: text})
			continue
		}
		target := forwardedContextTarget{
			ID:   normalizeForwardedString(entryMap["id"]),
			Name: normalizeForwardedString(entryMap["name"]),
		}
		if target.ID == "" {
			target.ID = normalizeForwardedString(entryMap["value"])
		}
		if target.Name == "" {
			target.Name = normalizeForwardedString(entryMap["label"])
		}
		if target.ID == "" && target.Name == "" {
			continue
		}
		if target.ID == "" {
			target.ID = target.Name
		}
		if target.Name == "" {
			target.Name = target.ID
		}
		key := target.ID + "|" + target.Name
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, target)
	}
	return result, nil
}

// normalizeForwardedString 将任意值转换为去空白字符串。
func normalizeForwardedString(raw any) string {
	text := strings.TrimSpace(fmt.Sprintf("%v", raw))
	if text == "<nil>" {
		return ""
	}
	return text
}

// findLastUserMessageTextAndIndex 返回最后一条 user 消息文本与索引。
func findLastUserMessageTextAndIndex(messages []types.Message) (string, int) {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != types.RoleUser {
			continue
		}
		return extractMessageTextContent(messages[i].Content), i
	}
	return "", -1
}

// extractMessageTextContent 提取 message.content 中的文本内容，不读取非文本字段。
func extractMessageTextContent(content any) string {
	switch value := content.(type) {
	case string:
		return strings.TrimSpace(value)
	case []types.InputContent:
		return joinInputTextContent(value)
	case []any:
		items := make([]types.InputContent, 0, len(value))
		for _, rawItem := range value {
			entryMap, err := normalizeForwardedPropsMap(rawItem)
			if err != nil || len(entryMap) == 0 {
				continue
			}
			itemType := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", entryMap["type"])))
			if itemType != types.InputContentTypeText {
				continue
			}
			items = append(items, types.InputContent{
				Type: itemType,
				Text: normalizeForwardedString(entryMap["text"]),
			})
		}
		return joinInputTextContent(items)
	default:
		return ""
	}
}

// joinInputTextContent 按顺序拼接文本 content fragment。
func joinInputTextContent(items []types.InputContent) string {
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		if strings.ToLower(strings.TrimSpace(item.Type)) != types.InputContentTypeText {
			continue
		}
		text := strings.TrimSpace(item.Text)
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// collectKnowledgeBaseNames 提取知识库名称，优先使用 ID，回退到 Name。
func collectKnowledgeBaseNames(items []forwardedContextTarget) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.ID)
		if name == "" {
			name = strings.TrimSpace(item.Name)
		}
		if name != "" {
			result = append(result, name)
		}
	}
	return util.NormalizeUniqueStringList(result)
}

// buildKnowledgeBaseContextMessage 构建注入模型上下文的知识库检索消息文本。
func buildKnowledgeBaseContextMessage(matches []dao.KnowledgeBaseSearchMatch) string {
	if len(matches) == 0 {
		return ""
	}
	lines := make([]string, 0, len(matches)*3+1)
	lines = append(lines, kbContextPrefix)
	for i, item := range matches {
		kbName := strings.TrimSpace(item.KnowledgeBaseName)
		if kbName == "" {
			kbName = "未命名知识库"
		}
		abstract := strings.TrimSpace(item.Abstract)
		if abstract == "" {
			abstract = "(无摘要)"
		}
		lines = append(lines, fmt.Sprintf("%d. [知识库:%s]", i+1, kbName))
		lines = append(lines, "uri: "+strings.TrimSpace(item.URI))
		lines = append(lines, "摘要: "+abstract)
	}
	return strings.Join(lines, "\n")
}

// buildSkillInstructionMessage 构建技能约束消息文本。
func buildSkillInstructionMessage(skills []forwardedContextTarget) string {
	labels := make([]string, 0, len(skills))
	for _, item := range skills {
		label := strings.TrimSpace(item.Name)
		if label == "" {
			label = strings.TrimSpace(item.ID)
		}
		if label != "" {
			labels = append(labels, label)
		}
	}
	labels = util.NormalizeUniqueStringList(labels)
	if len(labels) == 0 {
		return ""
	}
	return skillInstructionPrefix + "\n- " + strings.Join(labels, "\n- ")
}

// injectMessageBeforeUserAt 在指定 user 消息索引前插入一条临时消息。
func injectMessageBeforeUserAt(input *types.RunAgentInput, userIndex int, message types.Message) {
	if input == nil || strings.TrimSpace(fmt.Sprintf("%v", message.Content)) == "" {
		return
	}
	if userIndex < 0 || userIndex >= len(input.Messages) {
		input.Messages = append(input.Messages, message)
		return
	}
	next := make([]types.Message, 0, len(input.Messages)+1)
	next = append(next, input.Messages[:userIndex]...)
	next = append(next, message)
	next = append(next, input.Messages[userIndex:]...)
	input.Messages = next
}
