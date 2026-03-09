package agent

import (
	"context"
	"fmt"
	"openIntern/internal/config"
	"openIntern/internal/dao"
	builtinTool "openIntern/internal/services/builtin_tool"
	skillmiddleware "openIntern/internal/services/middlewares/skill"
	pluginsvc "openIntern/internal/services/plugin"
	"strings"

	"github.com/cloudwego/eino-ext/callbacks/apmplus"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/adk"
	einoSkill "github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/callbacks"
	einoTool "github.com/cloudwego/eino/components/tool"
)

// InitEino 初始化模型、工具、中间件及运行时依赖。
func (s *Service) InitEino(cfg config.LLMConfig, summaryCfg config.LLMConfig, toolsCfg config.ToolsConfig, apmCfg config.APMPlusConfig) (func(context.Context) error, error) {
	ctx := context.Background()

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

	var runtimeTitleModel *deepseek.ChatModel
	var err error
	if summaryCfg.APIKey != "" && summaryCfg.Model != "" {
		runtimeTitleModel, err = deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
			APIKey: summaryCfg.APIKey,
			Model:  summaryCfg.Model,
		})
		if err != nil {
			return nil, err
		}
	}

	sandboxBaseURL := strings.TrimSpace(toolsCfg.Sandbox.Url)
	if sandboxBaseURL == "" {
		return nil, fmt.Errorf("tools.sandbox.url is required")
	}
	pluginsvc.SetSandboxBaseURL(sandboxBaseURL)

	a2uiTools, err := builtinTool.GetA2UITools(ctx)
	if err != nil {
		return nil, err
	}
	cosTools, err := builtinTool.GetCOSTools(ctx)
	if err != nil {
		return nil, err
	}
	skillBackend, err := skillmiddleware.NewRemoteBackend(dao.SkillStore, s.deps.SkillFrontmatterStore)
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
	skillTools, err := skillmiddleware.GetSkillFileTools(dao.SkillStore)
	if err != nil {
		return nil, err
	}

	allTools := append([]einoTool.BaseTool{}, a2uiTools...)
	allTools = append(allTools, cosTools...)
	allTools = append(allTools, skillTools...)

	s.setState(runtimeState{
		apmplusShutdown:     shutdown,
		titleModel:          runtimeTitleModel,
		sandboxBaseURL:      sandboxBaseURL,
		agentTools:          allTools,
		agentMiddlewares:    []adk.AgentMiddleware{skillMiddleware},
		bootstrapChatConfig: cfg,
	})

	return shutdown, nil
}
