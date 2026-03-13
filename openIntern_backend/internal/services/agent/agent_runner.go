package agent

import (
	"context"
	"fmt"
	"log"
	"openIntern/internal/models"
	builtinTool "openIntern/internal/services/builtin_tool"
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

	sandboxTools, sandboxCleanup, err := builtinTool.GetSandboxMCPTools(ctx, state.sandboxBaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("load builtin sandbox tool failed: %w", err)
	}
	resolved = append(resolved, sandboxTools...)

	if runtimeConfig == nil {
		deduped, err := dedupeRuntimeToolsByName(ctx, resolved)
		if err != nil {
			if sandboxCleanup != nil {
				sandboxCleanup()
			}
			return nil, nil, err
		}
		return deduped, sandboxCleanup, nil
	}

	selectedToolIDs := runtimeConfig.Plugins.SelectedToolIDs
	if strings.EqualFold(runtimeConfig.Plugins.Mode, "search") {
		searchedToolIDs, err := pluginsvc.Plugin.SearchRuntimeToolIDs(ctx, runtimeConfig.Plugins.SearchQuery, pluginsvc.ToolSearchOptions{
			TopK:         runtimeConfig.Plugins.Search.TopK,
			RuntimeTypes: runtimeConfig.Plugins.Search.RuntimeTypes,
			MinScore:     runtimeConfig.Plugins.Search.MinScore,
			MaxMCPTools:  runtimeConfig.Plugins.Search.MaxMCPTools,
		})
		if err != nil {
			log.Printf("RunAgent tool search failed err=%v", err)
			return resolved, sandboxCleanup, nil
		}
		selectedToolIDs = searchedToolIDs
	}
	if len(selectedToolIDs) == 0 {
		deduped, err := dedupeRuntimeToolsByName(ctx, resolved)
		if err != nil {
			if sandboxCleanup != nil {
				sandboxCleanup()
			}
			return nil, nil, err
		}
		return deduped, sandboxCleanup, nil
	}

	codeTools, err := pluginsvc.Plugin.BuildRuntimeCodeTools(ctx, selectedToolIDs)
	if err != nil {
		return nil, nil, err
	}
	if len(codeTools) > 0 {
		resolved = append(resolved, codeTools...)
	}

	apiTools, err := pluginsvc.Plugin.BuildRuntimeAPITools(ctx, selectedToolIDs)
	if err != nil {
		return nil, nil, err
	}
	if len(apiTools) > 0 {
		resolved = append(resolved, apiTools...)
	}

	pluginTools, cleanup, err := pluginsvc.Plugin.BuildRuntimeMCPTools(ctx, selectedToolIDs)
	if err != nil {
		if sandboxCleanup != nil {
			sandboxCleanup()
		}
		if cleanup != nil {
			cleanup()
		}
		return nil, nil, err
	}
	if len(pluginTools) > 0 {
		resolved = append(resolved, pluginTools...)
	}
	deduped, err := dedupeRuntimeToolsByName(ctx, resolved)
	if err != nil {
		if sandboxCleanup != nil {
			sandboxCleanup()
		}
		if cleanup != nil {
			cleanup()
		}
		return nil, nil, err
	}
	return deduped, mergeToolCleanup(sandboxCleanup, cleanup), nil
}

// dedupeRuntimeToolsByName 按工具名去重，避免内建工具与插件广场同名工具重复注入。
func dedupeRuntimeToolsByName(ctx context.Context, tools []einoTool.BaseTool) ([]einoTool.BaseTool, error) {
	if len(tools) == 0 {
		return tools, nil
	}

	seen := make(map[string]struct{}, len(tools))
	result := make([]einoTool.BaseTool, 0, len(tools))
	for _, tool := range tools {
		info, err := tool.Info(ctx)
		if err != nil {
			return nil, err
		}
		if info == nil {
			return nil, fmt.Errorf("tool info is required")
		}
		name := strings.TrimSpace(info.Name)
		if name == "" {
			return nil, fmt.Errorf("tool name is required")
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, tool)
	}
	return result, nil
}

func mergeToolCleanup(cleanups ...func()) func() {
	return func() {
		for _, cleanup := range cleanups {
			if cleanup != nil {
				cleanup()
			}
		}
	}
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
