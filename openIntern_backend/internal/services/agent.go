package services

import (
	"context"
	"io"
	"openIntern/internal/config"
	agentpkg "openIntern/internal/services/agent"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
)

type modelCatalogResolverAdapter struct{}

// ResolveRuntimeSelection 将 services 层模型选择结果适配为 agent 子包结构。
func (modelCatalogResolverAdapter) ResolveRuntimeSelection(modelID, providerID string) (*agentpkg.RuntimeModelSelection, error) {
	selection, err := ModelCatalog.ResolveRuntimeSelection(modelID, providerID)
	if err != nil {
		return nil, err
	}
	if selection == nil {
		return nil, nil
	}
	return &agentpkg.RuntimeModelSelection{
		Provider: selection.Provider,
		Model:    selection.Model,
	}, nil
}

var defaultAgentService = agentpkg.NewService(agentpkg.Dependencies{
	A2UIService:                A2UI,
	FileUploader:               File,
	MessageStore:               Message,
	MemoryRetriever:            MemoryRetriever,
	MemorySyncStateStore:       MemorySyncState,
	MemoryUsageLogStore:        MemoryUsageLog,
	ThreadContextSnapshotStore: ThreadContextSnapshot,
	ThreadStore:                Thread,
	ModelCatalogResolver:       modelCatalogResolverAdapter{},
	ModelProviderResolver:      ModelProvider,
	SkillFrontmatterStore:      skillFrontmatterStore{},
})

// RunAgent 运行一个 Agent
func RunAgent(ctx context.Context, w io.Writer, input *types.RunAgentInput) error {
	return defaultAgentService.RunAgent(ctx, w, input)
}

// InitEino 初始化 agent 运行所需的模型、工具、中间件和上下文压缩策略。
func InitEino(cfg config.LLMConfig, summaryCfg config.LLMConfig, toolsCfg config.ToolsConfig, compressionCfg config.ContextCompressionConfig, apmCfg config.APMPlusConfig) (func(context.Context) error, error) {
	return defaultAgentService.InitEino(cfg, summaryCfg, toolsCfg, compressionCfg, apmCfg)
}
