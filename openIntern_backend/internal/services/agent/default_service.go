package agent

import (
	"context"
	"io"

	"openIntern/internal/config"
	a2uisvc "openIntern/internal/services/a2ui"
	chatsvc "openIntern/internal/services/chat"
	memorysvc "openIntern/internal/services/memory"
	modelsvc "openIntern/internal/services/model"
	skillsvc "openIntern/internal/services/skill"
	storagesvc "openIntern/internal/services/storage"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
)

type modelCatalogResolverAdapter struct{}

// ResolveRuntimeSelection adapts model service output to the agent runtime contract.
func (modelCatalogResolverAdapter) ResolveRuntimeSelection(modelID, providerID string) (*RuntimeModelSelection, error) {
	selection, err := modelsvc.ModelCatalog.ResolveRuntimeSelection(modelID, providerID)
	if err != nil {
		return nil, err
	}
	if selection == nil {
		return nil, nil
	}
	return &RuntimeModelSelection{
		Provider: selection.Provider,
		Model:    selection.Model,
	}, nil
}

var defaultService = NewService(Dependencies{
	A2UIService:                a2uisvc.A2UI,
	FileUploader:               storagesvc.File,
	MessageStore:               chatsvc.Message,
	MemoryRetriever:            memorysvc.MemoryRetriever,
	MemorySyncStateStore:       memorysvc.MemorySyncState,
	ThreadContextSnapshotStore: chatsvc.ThreadContextSnapshot,
	ThreadStore:                chatsvc.Thread,
	ModelCatalogResolver:       modelCatalogResolverAdapter{},
	ModelProviderResolver:      modelsvc.ModelProvider,
	SkillFrontmatterStore:      skillsvc.FrontmatterStoreAdapter{Store: skillsvc.SkillFrontmatter},
})

// RunAgent runs the default agent service entry used by HTTP handlers.
func RunAgent(ctx context.Context, w io.Writer, input *types.RunAgentInput) error {
	return defaultService.RunAgent(ctx, w, input)
}

// RunDebugAgent runs the transient editor-side agent debug entry.
func RunDebugAgent(ctx context.Context, w io.Writer, input *types.RunAgentInput) error {
	return defaultService.RunDebugAgent(ctx, w, input)
}

// InitEino initializes the default agent service runtime dependencies.
func InitEino(cfg config.LLMConfig, summaryCfg config.LLMConfig, toolsCfg config.ToolsConfig, agentCfg config.AgentConfig, compressionCfg config.ContextCompressionConfig, apmCfg config.APMPlusConfig) (func(context.Context) error, error) {
	return defaultService.InitEino(cfg, summaryCfg, toolsCfg, agentCfg, compressionCfg, apmCfg)
}
