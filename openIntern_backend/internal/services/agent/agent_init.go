package agent

import (
	"context"
	"fmt"
	"openIntern/internal/config"
	"openIntern/internal/dao"
	skillmiddleware "openIntern/internal/services/middlewares/skill"
	pluginsvc "openIntern/internal/services/plugin"
	"strings"

	"github.com/cloudwego/eino-ext/callbacks/apmplus"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/patchtoolcalls"
	einoSkill "github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/callbacks"
	einoModel "github.com/cloudwego/eino/components/model"
	einoTool "github.com/cloudwego/eino/components/tool"
)

// InitEino 初始化模型、工具、中间件、运行时依赖和上下文压缩参数。
func (s *Service) InitEino(summaryCfg config.LLMConfig, toolsCfg config.ToolsConfig, agentCfg config.AgentConfig, compressionCfg config.ContextCompressionConfig, apmCfg config.APMPlusConfig) (func(context.Context) error, error) {
	ctx := context.Background()
	compressionSettings := newContextCompressionSettings(compressionCfg)
	adk.SetLanguage(adk.LanguageChinese)

	shutdown := func(context.Context) error { return nil }
	if apmCfg.Host != "" && apmCfg.AppKey != "" && apmCfg.ServiceName != "" {
		cbh, apmShutdown, err := apmplus.NewApmplusHandler(&apmplus.Config{
			Host:        apmCfg.Host,
			AppKey:      apmCfg.AppKey,
			ServiceName: apmCfg.ServiceName,
			Release:     apmCfg.Release,
		})
		if err != nil {
			return nil, err
		}
		callbacks.AppendGlobalHandlers(cbh)
		shutdown = apmShutdown
	}

	var runtimeSummaryModel einoModel.ToolCallingChatModel
	var err error
	if compressionSettings.Enabled && (strings.TrimSpace(summaryCfg.APIKey) == "" || strings.TrimSpace(summaryCfg.Model) == "") {
		return nil, fmt.Errorf("summary_llm is required when context compression is enabled")
	}
	if strings.TrimSpace(summaryCfg.APIKey) != "" && strings.TrimSpace(summaryCfg.Model) != "" {
		runtimeSummaryModel, err = buildSummaryChatModel(ctx, summaryCfg)
		if err != nil {
			return nil, err
		}
	}

	sandboxBaseURL := strings.TrimSpace(toolsCfg.Sandbox.Url)
	if sandboxBaseURL == "" {
		return nil, fmt.Errorf("tools.sandbox.url is required")
	}
	pluginsvc.SetSandboxBaseURL(sandboxBaseURL)

	skillBackend, err := skillmiddleware.NewRemoteBackend(dao.SkillStore, s.deps.SkillFrontmatterStore)
	if err != nil {
		return nil, err
	}
	patchToolCallsMiddleware, err := patchtoolcalls.New(ctx, nil)
	if err != nil {
		return nil, err
	}
	skillMiddleware, err := einoSkill.NewMiddleware(ctx, &einoSkill.Config{
		Backend: skillBackend,
	})
	if err != nil {
		return nil, err
	}
	skillTools, err := skillmiddleware.GetSkillFileTools(dao.SkillStore)
	if err != nil {
		return nil, err
	}

	// A2UI/COS 已切换为内建插件显式绑定，这里只保留全局默认可用的 Skill 工具。
	allTools := append([]einoTool.BaseTool{}, skillTools...)

	s.setState(runtimeState{
		apmplusShutdown:    shutdown,
		summaryModel:       runtimeSummaryModel,
		sandboxBaseURL:     sandboxBaseURL,
		staticAgentTools:   allTools,
		agentHandlers:      []adk.ChatModelAgentMiddleware{patchToolCallsMiddleware, skillMiddleware},
		contextCompression: compressionSettings,
		maxIterations:      agentCfg.MaxIterations,
	})

	return shutdown, nil
}

// buildSummaryChatModel 根据配置构建摘要模型
func buildSummaryChatModel(ctx context.Context, cfg config.LLMConfig) (einoModel.ToolCallingChatModel, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	baseURL := strings.TrimSpace(cfg.BaseURL)

	switch provider {
	case "ark":
		return ark.NewChatModel(ctx, &ark.ChatModelConfig{
			APIKey:  cfg.APIKey,
			BaseURL: baseURL,
			Model:   cfg.Model,
		})
	case "openai":
		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
			APIKey:  cfg.APIKey,
			BaseURL: baseURL,
			Model:   cfg.Model,
		})
	case "deepseek", "":
		// 默认使用 deepseek
		return deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
			APIKey:  cfg.APIKey,
			BaseURL: baseURL,
			Model:   cfg.Model,
		})
	default:
		return nil, fmt.Errorf("unsupported summary_llm provider: %s", provider)
	}
}
