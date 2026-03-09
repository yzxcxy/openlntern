package agent

import (
	"context"
	"openIntern/internal/config"
	"openIntern/internal/models"
	builtinTool "openIntern/internal/services/builtin_tool"
	skillmiddleware "openIntern/internal/services/middlewares/skill"
	"sync"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/adk"
	einoTool "github.com/cloudwego/eino/components/tool"
)

type RuntimeModelSelection struct {
	Provider *models.ModelProvider
	Model    *models.ModelCatalog
}

type MessageStore interface {
	ListThreadMessages(threadID string) ([]models.Message, error)
	CreateMessages(messages []models.Message) error
}

type MemorySyncStateStore interface {
	ScheduleThreadSync(threadID, runID string) error
}

type MemoryUsageLogStore interface {
	RecordRunMemoryUsage(threadID, runID string, uris []string) error
}

type MemoryRetriever interface {
	BuildMemoryContext(ctx context.Context, input *types.RunAgentInput) (*types.Message, []string, error)
}

type ThreadContextSnapshotStore interface {
	GetLatestByThreadID(threadID string) (*models.ThreadContextSnapshot, error)
	Create(item *models.ThreadContextSnapshot) error
}

type ThreadStore interface {
	GetThreadByThreadID(threadID string) (*models.Thread, error)
	UpdateThreadTitle(threadID, title string) error
	TouchThread(threadID string) error
}

type ModelCatalogResolver interface {
	ResolveRuntimeSelection(modelID, providerID string) (*RuntimeModelSelection, error)
}

type ModelProviderKeyResolver interface {
	ResolveAPIKey(item *models.ModelProvider) (string, error)
}

type Dependencies struct {
	A2UIService                builtinTool.A2UIServiceInterface
	FileUploader               builtinTool.FileUploader
	MessageStore               MessageStore
	MemoryRetriever            MemoryRetriever
	MemorySyncStateStore       MemorySyncStateStore
	MemoryUsageLogStore        MemoryUsageLogStore
	ThreadContextSnapshotStore ThreadContextSnapshotStore
	ThreadStore                ThreadStore
	ModelCatalogResolver       ModelCatalogResolver
	ModelProviderResolver      ModelProviderKeyResolver
	SkillFrontmatterStore      skillmiddleware.SkillFrontmatterStore
}

type runtimeState struct {
	apmplusShutdown     func(context.Context) error
	titleModel          *deepseek.ChatModel
	sandboxBaseURL      string
	agentTools          []einoTool.BaseTool
	agentMiddlewares    []adk.AgentMiddleware
	bootstrapChatConfig config.LLMConfig
	contextCompression  contextCompressionSettings
}

type Service struct {
	deps  Dependencies
	mu    sync.RWMutex
	state runtimeState
}

// NewService 创建并返回 agent 子包的服务实例。
func NewService(deps Dependencies) *Service {
	return &Service{
		deps: deps,
		state: runtimeState{
			apmplusShutdown: func(context.Context) error { return nil },
		},
	}
}

// snapshotState 读取并拷贝当前运行态，避免调用方修改内部切片状态。
func (s *Service) snapshotState() runtimeState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := s.state
	if len(snapshot.agentTools) > 0 {
		snapshot.agentTools = append([]einoTool.BaseTool{}, snapshot.agentTools...)
	}
	if len(snapshot.agentMiddlewares) > 0 {
		snapshot.agentMiddlewares = append([]adk.AgentMiddleware{}, snapshot.agentMiddlewares...)
	}
	if snapshot.apmplusShutdown == nil {
		snapshot.apmplusShutdown = func(context.Context) error { return nil }
	}
	return snapshot
}

// setState 原子替换运行态配置，保证 shutdown 回调始终可调用。
func (s *Service) setState(state runtimeState) {
	if state.apmplusShutdown == nil {
		state.apmplusShutdown = func(context.Context) error { return nil }
	}
	s.mu.Lock()
	s.state = state
	s.mu.Unlock()
}
