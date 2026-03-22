package agent

import (
	"context"
	"fmt"
	"strings"

	"openIntern/internal/dao"
	skillmiddleware "openIntern/internal/services/middlewares/skill"
	pluginsvc "openIntern/internal/services/plugin"
	skillsvc "openIntern/internal/services/skill"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/patchtoolcalls"
	einoSkill "github.com/cloudwego/eino/adk/middlewares/skill"
	einoModel "github.com/cloudwego/eino/components/model"
	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

type compiledAgentModeRuntime struct {
	runner  *adk.Runner
	cleanup func()
}

type compiledAgentModeNode struct {
	agent   adk.Agent
	cleanup func()
}

type agentModeCompiler struct {
	service       *Service
	ctx           context.Context
	state         runtimeState
	ownerID       string
	runtimeConfig *AgentRuntimeConfig
	stack         []string
}

// buildAgentModeRunner compiles the selected agent tree into a streamable runner.
func (s *Service) buildAgentModeRunner(ctx context.Context, runtimeConfig *AgentRuntimeConfig, state runtimeState) (*compiledAgentModeRuntime, error) {
	if !isAgentConversationMode(runtimeConfig) {
		return nil, fmt.Errorf("agent mode is required")
	}
	ownerID := ownerIDFromContext(ctx)
	if ownerID == "" {
		return nil, fmt.Errorf("owner_id is required for agent mode")
	}
	selectedAgentID := selectedAgentIDFromRuntimeConfig(runtimeConfig)
	if selectedAgentID == "" {
		return nil, fmt.Errorf("selected_agent_id is required")
	}

	compiler := &agentModeCompiler{
		service:       s,
		ctx:           ctx,
		state:         state,
		ownerID:       ownerID,
		runtimeConfig: runtimeConfig,
	}
	node, err := compiler.compileExistingAgent(selectedAgentID, true, 1)
	if err != nil {
		return nil, err
	}
	return &compiledAgentModeRuntime{
		runner:  adk.NewRunner(ctx, adk.RunnerConfig{Agent: node.agent, EnableStreaming: true}),
		cleanup: node.cleanup,
	}, nil
}

// buildDebugAgentModeRunner compiles a transient editor definition into a streamable runner.
func (s *Service) buildDebugAgentModeRunner(ctx context.Context, runtimeConfig *AgentRuntimeConfig, state runtimeState, detail *AgentDetailView) (*compiledAgentModeRuntime, error) {
	if detail == nil {
		return nil, fmt.Errorf("debug agent detail is required")
	}
	ownerID := ownerIDFromContext(ctx)
	if ownerID == "" {
		return nil, fmt.Errorf("owner_id is required for agent mode")
	}
	compiler := &agentModeCompiler{
		service:       s,
		ctx:           ctx,
		state:         state,
		ownerID:       ownerID,
		runtimeConfig: runtimeConfig,
	}
	node, err := compiler.compileDetail(detail, true, 1, false)
	if err != nil {
		return nil, err
	}
	return &compiledAgentModeRuntime{
		runner:  adk.NewRunner(ctx, adk.RunnerConfig{Agent: node.agent, EnableStreaming: true}),
		cleanup: node.cleanup,
	}, nil
}

func (c *agentModeCompiler) compileExistingAgent(agentID string, allowRuntimeModelOverride bool, depth int) (*compiledAgentModeNode, error) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}
	detail, err := AgentDefinition.Get(c.ownerID, agentID)
	if err != nil {
		return nil, err
	}
	return c.compileDetail(detail, allowRuntimeModelOverride, depth, true)
}

func (c *agentModeCompiler) compileDetail(detail *AgentDetailView, allowRuntimeModelOverride bool, depth int, requireEnabled bool) (*compiledAgentModeNode, error) {
	if detail == nil {
		return nil, fmt.Errorf("agent detail is required")
	}
	agentID := strings.TrimSpace(detail.AgentID)
	if agentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}
	if depth > maxAgentCompileDepth {
		return nil, fmt.Errorf("agent dependency depth exceeds limit %d", maxAgentCompileDepth)
	}
	for _, stackID := range c.stack {
		if stackID == agentID {
			return nil, fmt.Errorf("agent dependency cycle detected at %s", agentID)
		}
	}
	if requireEnabled && !strings.EqualFold(detail.Status, AgentStatusEnabled) {
		return nil, fmt.Errorf("agent %s is not enabled", detail.Name)
	}
	if strings.EqualFold(detail.AgentType, AgentTypeSingle) && len(detail.SubAgentIDs) > 0 {
		return nil, fmt.Errorf("single agent cannot bind sub agents")
	}

	c.stack = append(c.stack, agentID)
	defer func() {
		c.stack = c.stack[:len(c.stack)-1]
	}()

	model, err := c.buildModel(detail, allowRuntimeModelOverride)
	if err != nil {
		return nil, err
	}
	tools, cleanup, err := c.buildTools(detail, depth)
	if err != nil {
		return nil, err
	}

	patchMiddleware, err := patchtoolcalls.New(c.ctx, nil)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, err
	}
	handlers := []adk.ChatModelAgentMiddleware{patchMiddleware}
	if skillMiddleware, err := c.buildSkillMiddleware(detail.SkillNames); err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, err
	} else if skillMiddleware != nil {
		handlers = append(handlers, skillMiddleware)
	}
	if resourceMiddleware := newAgentResourceInjectionMiddleware(detail.KnowledgeBaseNames, detail.AgentMemoryEnabled, c.service.deps.MemoryRetriever); resourceMiddleware != nil {
		handlers = append(handlers, resourceMiddleware)
	}

	agentNode, err := adk.NewChatModelAgent(c.ctx, &adk.ChatModelAgentConfig{
		Name:        buildCompiledAgentName(detail),
		Description: strings.TrimSpace(detail.Description),
		Instruction: strings.TrimSpace(detail.SystemPrompt),
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
			EmitInternalEvents: len(detail.SubAgentIDs) > 0,
		},
		Handlers: handlers,
	})
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, err
	}
	return &compiledAgentModeNode{
		agent:   agentNode,
		cleanup: cleanup,
	}, nil
}

func (c *agentModeCompiler) buildModel(detail *AgentDetailView, allowRuntimeModelOverride bool) (einoModel.ToolCallingChatModel, error) {
	if allowRuntimeModelOverride && c.runtimeConfig != nil && strings.TrimSpace(c.runtimeConfig.Model.ModelID) != "" {
		return c.service.buildRuntimeChatModel(c.ctx, c.runtimeConfig, c.state)
	}
	if strings.TrimSpace(detail.DefaultModelID) == "" {
		return c.service.buildBootstrapChatModel(c.ctx, c.state)
	}
	selection, err := c.service.deps.ModelCatalogResolver.ResolveRuntimeSelection(detail.DefaultModelID, "")
	if err != nil {
		return nil, err
	}
	if selection == nil {
		return c.service.buildBootstrapChatModel(c.ctx, c.state)
	}
	return c.service.buildChatModel(c.ctx, selection.Provider, selection.Model)
}

func (c *agentModeCompiler) buildTools(detail *AgentDetailView, depth int) ([]einoTool.BaseTool, func(), error) {
	resolvedTools, cleanup, err := c.buildBuiltinTools()
	if err != nil {
		return nil, nil, err
	}

	pluginTools, pluginCleanup, err := c.buildPluginTools(detail.ToolIDs)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, nil, err
	}
	cleanup = mergeToolCleanup(cleanup, pluginCleanup)
	resolvedTools = append(resolvedTools, pluginTools...)

	skillTools, err := c.buildSkillTools(detail.SkillNames)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, nil, err
	}
	resolvedTools = append(resolvedTools, skillTools...)

	subAgentTools, subAgentCleanup, err := c.buildSubAgentTools(detail.SubAgentIDs, depth)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, nil, err
	}
	cleanup = mergeToolCleanup(cleanup, subAgentCleanup)
	resolvedTools = append(resolvedTools, subAgentTools...)

	dedupedTools, err := dedupeRuntimeToolsByName(c.ctx, resolvedTools)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, nil, err
	}
	return dedupedTools, cleanup, nil
}

func (c *agentModeCompiler) buildBuiltinTools() ([]einoTool.BaseTool, func(), error) {
	// Agent 模式仅注入显式绑定资源（插件、skill、sub-agent）。
	// 不再默认注入内置工具，避免模型误调未绑定能力（如 upload_to_cos）。
	return nil, nil, nil
}

func (c *agentModeCompiler) buildPluginTools(toolIDs []string) ([]einoTool.BaseTool, func(), error) {
	if len(toolIDs) == 0 {
		return nil, nil, nil
	}
	codeTools, err := pluginsvc.Plugin.BuildRuntimeCodeTools(c.ctx, toolIDs)
	if err != nil {
		return nil, nil, err
	}
	apiTools, err := pluginsvc.Plugin.BuildRuntimeAPITools(c.ctx, toolIDs)
	if err != nil {
		return nil, nil, err
	}
	mcpTools, cleanup, err := pluginsvc.Plugin.BuildRuntimeMCPTools(c.ctx, toolIDs)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, nil, err
	}
	tools := make([]einoTool.BaseTool, 0, len(codeTools)+len(apiTools)+len(mcpTools))
	tools = append(tools, codeTools...)
	tools = append(tools, apiTools...)
	tools = append(tools, mcpTools...)
	return tools, cleanup, nil
}

func (c *agentModeCompiler) buildSkillMiddleware(skillNames []string) (adk.ChatModelAgentMiddleware, error) {
	if len(skillNames) == 0 {
		return nil, nil
	}
	repo, store, err := c.buildScopedSkillBackend(skillNames)
	if err != nil {
		return nil, err
	}
	backend, err := skillmiddleware.NewRemoteBackend(repo, store)
	if err != nil {
		return nil, err
	}
	return einoSkill.NewMiddleware(c.ctx, &einoSkill.Config{Backend: backend})
}

func (c *agentModeCompiler) buildSkillTools(skillNames []string) ([]einoTool.BaseTool, error) {
	if len(skillNames) == 0 {
		return nil, nil
	}
	repo, _, err := c.buildScopedSkillBackend(skillNames)
	if err != nil {
		return nil, err
	}
	return skillmiddleware.GetSkillFileTools(repo)
}

func (c *agentModeCompiler) buildScopedSkillBackend(skillNames []string) (skillmiddleware.SkillRepository, skillmiddleware.SkillFrontmatterStore, error) {
	repo, err := skillmiddleware.NewScopedRepository(dao.SkillStore, skillNames)
	if err != nil {
		return nil, nil, err
	}
	store, err := skillmiddleware.NewScopedFrontmatterStore(skillsvc.FrontmatterStoreAdapter{Store: skillsvc.SkillFrontmatter}, skillNames)
	if err != nil {
		return nil, nil, err
	}
	return repo, store, nil
}

func (c *agentModeCompiler) buildSubAgentTools(subAgentIDs []string, depth int) ([]einoTool.BaseTool, func(), error) {
	if len(subAgentIDs) == 0 {
		return nil, nil, nil
	}
	tools := make([]einoTool.BaseTool, 0, len(subAgentIDs))
	cleanups := make([]func(), 0, len(subAgentIDs))
	for _, subAgentID := range subAgentIDs {
		node, err := c.compileExistingAgent(subAgentID, false, depth+1)
		if err != nil {
			mergeToolCleanup(cleanups...)()
			return nil, nil, err
		}
		tools = append(tools, adk.NewAgentTool(c.ctx, node.agent))
		cleanups = append(cleanups, node.cleanup)
	}
	return tools, mergeToolCleanup(cleanups...), nil
}

func buildCompiledAgentName(detail *AgentDetailView) string {
	name := strings.TrimSpace(detail.Name)
	if name != "" {
		return name
	}
	return strings.TrimSpace(detail.AgentID)
}
