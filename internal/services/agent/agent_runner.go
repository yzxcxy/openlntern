package agent

import (
	"context"
	"fmt"
	"openIntern/internal/models"
	pluginsvc "openIntern/internal/services/plugin"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	einoModel "github.com/cloudwego/eino/components/model"
	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// buildEinoRunner 基于当前运行时配置组装可流式执行的 runner。
func (s *Service) buildEinoRunner(ctx context.Context, runtimeConfig *AgentRuntimeConfig, state runtimeState) (*adk.Runner, func(), error) {
	chatModel, err := s.buildRuntimeChatModel(ctx, runtimeConfig, state)
	if err != nil {
		return nil, nil, err
	}
	runtimeTools, cleanup, err := s.resolveRuntimeTools(ctx, runtimeConfig, state)
	if err != nil {
		return nil, nil, err
	}
	agent := "openintern agent"
	agentNode, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "openintern_agent",
		Description: agent,
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: runtimeTools,
			},
		},
		Middlewares: state.agentMiddlewares,
	})
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, nil, err
	}
	return adk.NewRunner(ctx, adk.RunnerConfig{Agent: agentNode, EnableStreaming: true}), cleanup, nil
}

// resolveRuntimeTools 解析内置工具与插件工具，并返回必要的清理函数。
func (s *Service) resolveRuntimeTools(ctx context.Context, runtimeConfig *AgentRuntimeConfig, state runtimeState) ([]einoTool.BaseTool, func(), error) {
	resolved := make([]einoTool.BaseTool, 0, len(state.agentTools))
	resolved = append(resolved, state.agentTools...)

	if runtimeConfig == nil || strings.EqualFold(runtimeConfig.Plugins.Mode, "search") {
		return resolved, nil, nil
	}

	codeTools, err := pluginsvc.Plugin.BuildRuntimeCodeTools(ctx, runtimeConfig.Plugins.SelectedToolIDs)
	if err != nil {
		return nil, nil, err
	}
	if len(codeTools) > 0 {
		resolved = append(resolved, codeTools...)
	}

	apiTools, err := pluginsvc.Plugin.BuildRuntimeAPITools(ctx, runtimeConfig.Plugins.SelectedToolIDs)
	if err != nil {
		return nil, nil, err
	}
	if len(apiTools) > 0 {
		resolved = append(resolved, apiTools...)
	}

	pluginTools, cleanup, err := pluginsvc.Plugin.BuildRuntimeMCPTools(ctx, runtimeConfig.Plugins.SelectedToolIDs)
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

// buildRuntimeChatModel 优先使用 runtime 指定模型，否则回退到默认模型。
func (s *Service) buildRuntimeChatModel(ctx context.Context, runtimeConfig *AgentRuntimeConfig, state runtimeState) (einoModel.ToolCallingChatModel, error) {
	if runtimeConfig != nil {
		selection, err := s.deps.ModelCatalogResolver.ResolveRuntimeSelection(runtimeConfig.Model.ModelID, runtimeConfig.Model.ProviderID)
		if err != nil {
			return nil, err
		}
		if selection != nil {
			return s.buildChatModel(ctx, selection.Provider, selection.Model)
		}
	}
	return s.buildBootstrapChatModel(ctx, state)
}

// buildBootstrapChatModel 使用系统默认配置构建聊天模型。
func (s *Service) buildBootstrapChatModel(ctx context.Context, state runtimeState) (einoModel.ToolCallingChatModel, error) {
	if strings.TrimSpace(state.bootstrapChatConfig.APIKey) == "" || strings.TrimSpace(state.bootstrapChatConfig.Model) == "" {
		return nil, fmt.Errorf("no default chat model configured")
	}
	return ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey: state.bootstrapChatConfig.APIKey,
		Model:  state.bootstrapChatConfig.Model,
	})
}

// buildChatModel 根据 provider 类型构建对应的底层聊天模型实现。
func (s *Service) buildChatModel(ctx context.Context, provider *models.ModelProvider, modelItem *models.ModelCatalog) (einoModel.ToolCallingChatModel, error) {
	if provider == nil || modelItem == nil {
		return nil, fmt.Errorf("provider and model are required")
	}
	apiKey, err := s.deps.ModelProviderResolver.ResolveAPIKey(provider)
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
	case "deepseek":
		return deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
			APIKey:  apiKey,
			BaseURL: strings.TrimSpace(provider.BaseURL),
			Model:   strings.TrimSpace(modelItem.ModelKey),
		})
	default:
		return nil, fmt.Errorf("unsupported api_type: %s", provider.APIType)
	}
}
