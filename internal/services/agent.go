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
	skillmiddleware "openIntern/internal/services/middlewares/skill"
	"openIntern/internal/services/tools"
	"sort"
	"strings"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/cloudwego/eino-ext/callbacks/apmplus"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	einoSkill "github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/callbacks"
	einoModel "github.com/cloudwego/eino/components/model"
	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

// RunAgent 运行一个 Agent
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
	if runID == "" {
		return fmt.Errorf("run_id is required")
	}

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
	runtimeConfig, err := applyForwardedPropsChain(mergedInput)
	if err != nil {
		_ = s.Error(err.Error(), "forwarded_props_handle_failed")
		log.Printf("RunAgent forwarded props handle failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	if runtimeConfig != nil && strings.EqualFold(runtimeConfig.Conversation.Mode, "agent") {
		err := fmt.Errorf("agent mode is not available yet")
		_ = s.Error(err.Error(), "agent_mode_not_available")
		return err
	}
	einoMessages, err := agui.AGUIRunInputToEinoMessages(mergedInput)
	if err != nil {
		_ = s.Error(err.Error(), "agui_to_eino_failed")
		log.Printf("RunAgent agui to eino failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	// 注入 A2UI 工具所需的 context：service 与 sender
	ctx = context.WithValue(ctx, tools.ContextKeyA2UIService, A2UI)
	ctx = context.WithValue(ctx, tools.ContextKeyA2UISender, s)
	ctx = context.WithValue(ctx, tools.ContextKeyFileUploader, File)
	ctx = context.WithValue(ctx, tools.ContextKeySandboxBaseURL, sandboxBaseURL)
	err = runEinoStreaming(ctx, s, einoMessages, runtimeConfig)
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
	if err := persistUserLastMessage(threadID, runID, input.Messages); err != nil {
		log.Printf("RunAgent persist user message failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	if err := persistAccumulatedMessages(threadID, runID, flushed); err != nil {
		log.Printf("RunAgent persist failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	if err := Thread.TouchThread(threadID); err != nil {
		log.Printf("RunAgent touch thread failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
		return err
	}
	if err := ensureThreadTitle(ctx, threadID, input.Messages); err != nil {
		log.Printf("RunAgent generate title failed thread_id=%s run_id=%s err=%v", threadID, runID, err)
	}

	return nil
}

var apmplusShutdown func(context.Context) error
var titleModel *deepseek.ChatModel
var sandboxBaseURL string
var agentTools []einoTool.BaseTool
var agentMiddlewares []adk.AgentMiddleware
var bootstrapChatConfig config.LLMConfig

type openVikingSkillClient struct{}

func (openVikingSkillClient) SkillsRoot() string {
	return OpenViking.SkillsRoot()
}

func (openVikingSkillClient) List(ctx context.Context, uri string, recursive bool) ([]map[string]any, error) {
	return OpenVikingList(ctx, uri, recursive)
}

func (openVikingSkillClient) ReadAbstract(ctx context.Context, uri string) (string, error) {
	return OpenVikingReadAbstract(ctx, uri)
}

func (openVikingSkillClient) ReadContent(ctx context.Context, uri string) (string, error) {
	return OpenVikingReadContent(ctx, uri)
}

type skillFrontmatterStore struct{}

func (skillFrontmatterStore) ListByNames(names []string) ([]skillmiddleware.SkillFrontmatterRecord, error) {
	items, err := SkillFrontmatter.ListByNames(names)
	if err != nil {
		return nil, err
	}
	records := make([]skillmiddleware.SkillFrontmatterRecord, 0, len(items))
	for _, item := range items {
		records = append(records, skillmiddleware.SkillFrontmatterRecord{
			SkillName: item.SkillName,
			Raw:       item.Raw,
		})
	}
	return records, nil
}

func (skillFrontmatterStore) GetByName(name string) (*skillmiddleware.SkillFrontmatterRecord, error) {
	item, err := SkillFrontmatter.GetByName(name)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	return &skillmiddleware.SkillFrontmatterRecord{
		SkillName: item.SkillName,
		Raw:       item.Raw,
	}, nil
}

func InitEino(cfg config.LLMConfig, summaryCfg config.LLMConfig, toolsCfg config.ToolsConfig, apmCfg config.APMPlusConfig) (func(context.Context) error, error) {
	ctx := context.Background()
	var err error
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
	bootstrapChatConfig = cfg
	titleModel = nil
	if summaryCfg.APIKey != "" && summaryCfg.Model != "" {
		titleModel, err = deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
			APIKey: summaryCfg.APIKey,
			Model:  summaryCfg.Model,
		})
		if err != nil {
			return nil, err
		}
	}
	sandboxBaseURL = strings.TrimSpace(toolsCfg.Sandbox.Url)
	if sandboxBaseURL == "" {
		return nil, fmt.Errorf("tools.sandbox.url is required")
	}
	a2uiTools, err := tools.GetA2UITools(ctx)
	if err != nil {
		return nil, err
	}
	cosTools, err := tools.GetCOSTools(ctx)
	if err != nil {
		return nil, err
	}
	skillBackend, err := skillmiddleware.NewOpenVikingBackend(openVikingSkillClient{}, skillFrontmatterStore{})
	if err != nil {
		return nil, err
	}
	skillMiddleware, err := einoSkill.New(ctx, &einoSkill.Config{
		Backend:    skillBackend,
		UseChinese: true,
	})
	if err != nil {
		return nil, err
	}
	skillTools, err := skillmiddleware.GetSkillFileTools(openVikingSkillClient{})
	if err != nil {
		return nil, err
	}
	allTools := append([]einoTool.BaseTool{}, a2uiTools...)
	allTools = append(allTools, cosTools...)
	allTools = append(allTools, skillTools...)
	agentTools = allTools
	agentMiddlewares = []adk.AgentMiddleware{skillMiddleware}
	return apmplusShutdown, nil
}

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

func normalizeForwardedPropsStringList(raw any) ([]string, error) {
	if raw == nil {
		return nil, nil
	}
	if values, ok := raw.([]string); ok {
		return normalizeUniqueStringList(values), nil
	}
	if values, ok := raw.([]any); ok {
		result := make([]string, 0, len(values))
		for _, value := range values {
			text := strings.TrimSpace(fmt.Sprintf("%v", value))
			if text != "" {
				result = append(result, text)
			}
		}
		return normalizeUniqueStringList(result), nil
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
		return normalizeUniqueStringList(result), nil
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

func runEinoStreaming(ctx context.Context, sender *agui.AccumulatingSender, messages []*schema.Message, runtimeConfig *AgentRuntimeConfig) error {
	if len(messages) == 0 {
		return nil
	}
	runner, cleanup, err := buildEinoRunner(ctx, runtimeConfig)
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

func buildEinoRunner(ctx context.Context, runtimeConfig *AgentRuntimeConfig) (*adk.Runner, func(), error) {
	chatModel, err := buildRuntimeChatModel(ctx, runtimeConfig)
	if err != nil {
		return nil, nil, err
	}
	runtimeTools, cleanup, err := resolveRuntimeTools(ctx, runtimeConfig)
	if err != nil {
		return nil, nil, err
	}
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "openintern_agent",
		Description: "openintern agent",
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: runtimeTools,
			},
		},
		Middlewares: agentMiddlewares,
	})
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, nil, err
	}
	return adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent, EnableStreaming: true}), cleanup, nil
}

func resolveRuntimeTools(ctx context.Context, runtimeConfig *AgentRuntimeConfig) ([]einoTool.BaseTool, func(), error) {
	resolved := make([]einoTool.BaseTool, 0, len(agentTools))
	resolved = append(resolved, agentTools...)

	if runtimeConfig == nil || strings.EqualFold(runtimeConfig.Plugins.Mode, "search") {
		return resolved, nil, nil
	}

	codeTools, err := Plugin.BuildRuntimeCodeTools(ctx, runtimeConfig.Plugins.SelectedToolIDs)
	if err != nil {
		return nil, nil, err
	}
	if len(codeTools) > 0 {
		resolved = append(resolved, codeTools...)
	}

	apiTools, err := Plugin.BuildRuntimeAPITools(ctx, runtimeConfig.Plugins.SelectedToolIDs)
	if err != nil {
		return nil, nil, err
	}
	if len(apiTools) > 0 {
		resolved = append(resolved, apiTools...)
	}

	pluginTools, cleanup, err := Plugin.BuildRuntimeMCPTools(ctx, runtimeConfig.Plugins.SelectedToolIDs)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, nil, err
	}
	if len(pluginTools) > 0 {
		resolved = append(resolved, pluginTools...)
	}
	return resolved, cleanup, nil
}

func buildRuntimeChatModel(ctx context.Context, runtimeConfig *AgentRuntimeConfig) (einoModel.ToolCallingChatModel, error) {
	if runtimeConfig != nil {
		selection, err := ModelCatalog.ResolveRuntimeSelection(runtimeConfig.Model.ModelID, runtimeConfig.Model.ProviderID)
		if err != nil {
			return nil, err
		}
		if selection != nil {
			return buildChatModel(ctx, selection.Provider, selection.Model)
		}
	}
	return buildBootstrapChatModel(ctx)
}

func buildBootstrapChatModel(ctx context.Context) (einoModel.ToolCallingChatModel, error) {
	if strings.TrimSpace(bootstrapChatConfig.APIKey) == "" || strings.TrimSpace(bootstrapChatConfig.Model) == "" {
		return nil, fmt.Errorf("no default chat model configured")
	}
	return ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey: bootstrapChatConfig.APIKey,
		Model:  bootstrapChatConfig.Model,
	})
}

func buildChatModel(ctx context.Context, provider *models.ModelProvider, modelItem *models.ModelCatalog) (einoModel.ToolCallingChatModel, error) {
	if provider == nil || modelItem == nil {
		return nil, fmt.Errorf("provider and model are required")
	}
	apiKey, err := ModelProvider.ResolveAPIKey(provider)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(provider.APIType)) {
	case "ark":
		return ark.NewChatModel(ctx, &ark.ChatModelConfig{
			APIKey:  apiKey,
			BaseURL: strings.TrimSpace(provider.BaseURL),
			Model:   strings.TrimSpace(modelItem.ModelKey),
		})
	case "openai":
		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
			APIKey:  apiKey,
			BaseURL: strings.TrimSpace(provider.BaseURL),
			Model:   strings.TrimSpace(modelItem.ModelKey),
		})
	default:
		return nil, fmt.Errorf("unsupported api_type: %s", provider.APIType)
	}
}

func ensureThreadTitle(ctx context.Context, threadID string, messages []types.Message) error {
	if threadID == "" {
		return nil
	}
	thread, err := Thread.GetThreadByThreadID(threadID)
	if err != nil || thread == nil {
		return err
	}
	if strings.TrimSpace(thread.Title) != "" {
		return nil
	}
	source := extractTitleSource(messages)
	if source == "" {
		return nil
	}
	title, err := generateTitle(ctx, source)
	if err != nil {
		return err
	}
	title = sanitizeTitle(title)
	if title == "" {
		return nil
	}
	return Thread.UpdateThreadTitle(thread.ThreadID, title)
}

func extractTitleSource(messages []types.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role == types.RoleUser {
			switch v := msg.Content.(type) {
			case string:
				return strings.TrimSpace(v)
			case nil:
				return ""
			default:
				return strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		}
	}
	return ""
}

func generateTitle(ctx context.Context, content string) (string, error) {
	if titleModel == nil {
		return "", nil
	}
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", nil
	}
	if len(trimmed) > 600 {
		trimmed = trimmed[:600]
	}
	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: "请为用户对话生成简短中文标题，不超过20字，只输出标题本身。",
		},
		{
			Role:    schema.User,
			Content: trimmed,
		},
	}
	resp, err := titleModel.Generate(ctx, messages)
	if err != nil || resp == nil {
		return "", err
	}
	return resp.Content, nil
}

func sanitizeTitle(title string) string {
	cleaned := strings.TrimSpace(title)
	cleaned = strings.Trim(cleaned, "“”\"'")
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return ""
	}
	runes := []rune(cleaned)
	if len(runes) > 40 {
		return string(runes[:40])
	}
	return cleaned
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

func persistUserLastMessage(threadID, runID string, messages []types.Message) error {
	if len(messages) == 0 {
		return nil
	}
	if runID == "" {
		return fmt.Errorf("run_id is required")
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
	msgID := normalizeMsgID(lastUser.ID)
	normalized := *lastUser
	normalized.ID = msgID
	b, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	return Message.CreateMessages([]models.Message{
		{
			MsgID:    msgID,
			ThreadID: threadID,
			RunID:    runID,
			Type:     "text",
			Content:  string(b),
			Status:   "completed",
		},
	})
}

func persistAccumulatedMessages(threadID, runID string, messages []agui.AccumulatedMessage) error {
	if len(messages) == 0 {
		return nil
	}
	if runID == "" {
		return fmt.Errorf("run_id is required")
	}
	modelsMessages := make([]models.Message, 0, len(messages))
	for _, msg := range messages {
		msg.MsgID = normalizeMsgID(msg.MsgID)
		content, metadata := buildMessageContentAndMetadata(msg)
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
	return Message.CreateMessages(modelsMessages)
}

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

func buildMessageContentAndMetadata(msg agui.AccumulatedMessage) (string, string) {
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
