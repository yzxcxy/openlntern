package agent

import (
	"context"
	"fmt"
	"log"
	"openIntern/internal/models"
	builtinTool "openIntern/internal/services/builtin_tool"
	toolsearchmiddleware "openIntern/internal/services/middlewares/toolsearch"
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

type runtimeToolSet struct {
	staticTools              []einoTool.BaseTool
	dynamicTools             []einoTool.BaseTool
	initialVisibleToolNames  []string
	allowToolSearchSelection bool
	toolVisibilityMiddleware adk.ChatModelAgentMiddleware
	cleanup                  func()
}

// buildEinoRunner 基于当前运行时配置组装可流式执行的 runner。
func (s *Service) buildEinoRunner(ctx context.Context, runtimeConfig *AgentRuntimeConfig, state runtimeState) (*adk.Runner, func(), error) {
	chatModel, err := s.buildRuntimeChatModel(ctx, runtimeConfig, state)
	if err != nil {
		return nil, nil, err
	}
	runtimeTools, err := s.resolveRuntimeToolSet(ctx, runtimeConfig, state)
	if err != nil {
		return nil, nil, err
	}

	handlers := append([]adk.ChatModelAgentMiddleware{}, state.agentHandlers...)
	if runtimeTools.toolVisibilityMiddleware != nil {
		handlers = append(handlers, runtimeTools.toolVisibilityMiddleware)
	}
	agent := "openintern agent"
	agentNode, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "openintern_agent",
		Description:   agent,
		Model:         chatModel,
		MaxIterations: state.maxIterations,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: runtimeTools.staticTools,
			},
		},
		Handlers: handlers,
	})
	if err != nil {
		if runtimeTools.cleanup != nil {
			runtimeTools.cleanup()
		}
		return nil, nil, err
	}
	return adk.NewRunner(ctx, adk.RunnerConfig{Agent: agentNode, EnableStreaming: true}), runtimeTools.cleanup, nil
}

// resolveRuntimeToolSet 解析静态工具、动态工具与模型可见性控制中间件。
func (s *Service) resolveRuntimeToolSet(ctx context.Context, runtimeConfig *AgentRuntimeConfig, state runtimeState) (*runtimeToolSet, error) {
	resolved := &runtimeToolSet{
		staticTools: append([]einoTool.BaseTool{}, state.staticAgentTools...),
	}

	sandboxTools, sandboxCleanup, err := builtinTool.GetSandboxMCPTools(ctx, state.sandboxBaseURL)
	if err != nil {
		return nil, fmt.Errorf("load builtin sandbox tool failed: %w", err)
	}
	resolved.cleanup = sandboxCleanup
	resolved.staticTools = append(resolved.staticTools, sandboxTools...)
	if runtimeConfig != nil && len(runtimeConfig.KnowledgeBaseNames) > 0 {
		knowledgeBaseTools, err := builtinTool.GetKnowledgeBaseTools(ctx, runtimeConfig.KnowledgeBaseNames)
		if err != nil {
			if resolved.cleanup != nil {
				resolved.cleanup()
			}
			return nil, fmt.Errorf("load builtin knowledge base tool failed: %w", err)
		}
		resolved.staticTools = append(resolved.staticTools, knowledgeBaseTools...)
	}

	if runtimeConfig == nil {
		return finalizeRuntimeToolSet(ctx, resolved)
	}

	mode := strings.ToLower(strings.TrimSpace(runtimeConfig.Plugins.Mode))
	switch mode {
	case "search":
		searchTool, err := toolsearchmiddleware.NewTool(ctx)
		if err != nil {
			if resolved.cleanup != nil {
				resolved.cleanup()
			}
			return nil, err
		}
		resolved.staticTools = append(resolved.staticTools, searchTool)
		resolved.allowToolSearchSelection = true
	case "select":
		if len(runtimeConfig.Plugins.SelectedToolIDs) == 0 {
			return finalizeRuntimeToolSet(ctx, resolved)
		}
		// 内建插件不进入动态搜索池，只在显式选中后直接挂到静态工具集合中。
		builtinTools, err := pluginsvc.Plugin.BuildRuntimeBuiltinTools(ctx, runtimeConfig.Plugins.SelectedToolIDs)
		if err != nil {
			if resolved.cleanup != nil {
				resolved.cleanup()
			}
			return nil, err
		}
		resolved.staticTools = append(resolved.staticTools, builtinTools...)
	default:
		return finalizeRuntimeToolSet(ctx, resolved)
	}

	dynamicTools, dynamicCleanup, err := s.resolveDynamicRuntimeTools(ctx)
	if err != nil {
		if resolved.cleanup != nil {
			resolved.cleanup()
		}
		return nil, err
	}
	resolved.cleanup = mergeToolCleanup(resolved.cleanup, dynamicCleanup)
	resolved.dynamicTools = dynamicTools

	if mode == "select" {
		initialVisibleToolNames, err := pluginsvc.Plugin.ResolveEnabledRuntimeToolNamesByIDs(runtimeConfig.Plugins.SelectedToolIDs)
		if err != nil {
			if resolved.cleanup != nil {
				resolved.cleanup()
			}
			return nil, err
		}
		resolved.initialVisibleToolNames = initialVisibleToolNames
	}

	return finalizeRuntimeToolSet(ctx, resolved)
}

// resolveDynamicRuntimeTools 预构建全部启用态插件工具，供运行时按可见性过滤。
func (s *Service) resolveDynamicRuntimeTools(ctx context.Context) ([]einoTool.BaseTool, func(), error) {
	codeTools, err := pluginsvc.Plugin.BuildAllRuntimeCodeTools(ctx)
	if err != nil {
		return nil, nil, err
	}

	apiTools, err := pluginsvc.Plugin.BuildAllRuntimeAPITools(ctx)
	if err != nil {
		return nil, nil, err
	}

	mcpTools, cleanup, err := pluginsvc.Plugin.BuildAllRuntimeMCPTools(ctx)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, nil, err
	}

	resolved := make([]einoTool.BaseTool, 0, len(codeTools)+len(apiTools)+len(mcpTools))
	resolved = append(resolved, codeTools...)
	resolved = append(resolved, apiTools...)
	resolved = append(resolved, mcpTools...)
	return resolved, cleanup, nil
}

// finalizeRuntimeToolSet 对静态/动态工具做名称去重，并在需要时挂上可见性 middleware。
func finalizeRuntimeToolSet(ctx context.Context, resolved *runtimeToolSet) (*runtimeToolSet, error) {
	staticTools, dynamicTools, err := dedupeRuntimeToolSetsByName(ctx, resolved.staticTools, resolved.dynamicTools)
	if err != nil {
		if resolved.cleanup != nil {
			resolved.cleanup()
		}
		return nil, err
	}
	resolved.staticTools = staticTools
	resolved.dynamicTools = dynamicTools

	if len(resolved.dynamicTools) == 0 {
		return resolved, nil
	}

	toolVisibilityMiddleware, err := toolsearchmiddleware.NewVisibilityMiddleware(ctx, resolved.dynamicTools, resolved.initialVisibleToolNames, resolved.allowToolSearchSelection)
	if err != nil {
		if resolved.cleanup != nil {
			resolved.cleanup()
		}
		return nil, err
	}
	resolved.toolVisibilityMiddleware = toolVisibilityMiddleware
	return resolved, nil
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

// dedupeRuntimeToolSetsByName 在保留静态工具优先级的前提下过滤动态工具重名项。
func dedupeRuntimeToolSetsByName(ctx context.Context, staticTools []einoTool.BaseTool, dynamicTools []einoTool.BaseTool) ([]einoTool.BaseTool, []einoTool.BaseTool, error) {
	dedupedStatic, err := dedupeRuntimeToolsByName(ctx, staticTools)
	if err != nil {
		return nil, nil, err
	}
	if len(dynamicTools) == 0 {
		return dedupedStatic, nil, nil
	}

	staticNames := make(map[string]struct{}, len(dedupedStatic))
	for _, tool := range dedupedStatic {
		info, err := tool.Info(ctx)
		if err != nil {
			return nil, nil, err
		}
		if info == nil {
			return nil, nil, fmt.Errorf("tool info is required")
		}
		staticNames[strings.TrimSpace(info.Name)] = struct{}{}
	}

	dedupedDynamic, err := dedupeRuntimeToolsByName(ctx, dynamicTools)
	if err != nil {
		return nil, nil, err
	}
	filteredDynamic := make([]einoTool.BaseTool, 0, len(dedupedDynamic))
	for _, tool := range dedupedDynamic {
		info, err := tool.Info(ctx)
		if err != nil {
			return nil, nil, err
		}
		if info == nil {
			return nil, nil, fmt.Errorf("tool info is required")
		}
		if _, exists := staticNames[strings.TrimSpace(info.Name)]; exists {
			log.Printf("RunAgent skip dynamic tool because of duplicate name tool_name=%s", info.Name)
			continue
		}
		filteredDynamic = append(filteredDynamic, tool)
	}
	return dedupedStatic, filteredDynamic, nil
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
