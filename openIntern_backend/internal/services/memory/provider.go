package memory

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"openIntern/internal/config"
	"openIntern/internal/models"
	"openIntern/internal/services/memory/contracts"
	openvikingmemory "openIntern/internal/services/memory/openviking"
)

// RetrieverBackend defines the provider-specific retrieval contract for long-term memory injection.
type RetrieverBackend interface {
	Configured() bool
	Retrieve(ctx context.Context, userID string, inputText string) ([]contracts.RetrievedMemory, error)
}

// SyncBackend defines the provider-specific synchronization contract for long-term memory persistence.
type SyncBackend interface {
	Configured() bool
	SyncDelay() time.Duration
	SyncPollInterval() time.Duration
	SyncTimeout() time.Duration
	SyncRetryDelay() time.Duration
	SubmitMessages(ctx context.Context, state models.MemorySyncState, messages []contracts.SyncMessage) (operationID string, err error)
	PollOperation(ctx context.Context, operationID string) (status string, operationErr string, err error)
}

type noopRetrieverBackend struct{}

func (noopRetrieverBackend) Configured() bool {
	return false
}

func (noopRetrieverBackend) Retrieve(context.Context, string, string) ([]contracts.RetrievedMemory, error) {
	return nil, nil
}

type noopSyncBackend struct{}

func (noopSyncBackend) Configured() bool {
	return false
}

func (noopSyncBackend) SyncDelay() time.Duration {
	return defaultMemorySyncDelay
}

func (noopSyncBackend) SyncPollInterval() time.Duration {
	return defaultMemorySyncPollInterval
}

func (noopSyncBackend) SyncTimeout() time.Duration {
	return defaultMemorySyncTimeout
}

func (noopSyncBackend) SyncRetryDelay() time.Duration {
	return defaultMemorySyncRetryDelay
}

func (noopSyncBackend) SubmitMessages(context.Context, models.MemorySyncState, []contracts.SyncMessage) (string, error) {
	return "", nil
}

func (noopSyncBackend) PollOperation(context.Context, string) (string, string, error) {
	return "", "", nil
}

var (
	memoryProviderMu  sync.RWMutex
	activeRetriever   RetrieverBackend = noopRetrieverBackend{}
	activeSyncBackend SyncBackend      = noopSyncBackend{}
)

// InitMemory selects and initializes the active long-term memory provider.
func InitMemory(cfg config.ToolsConfig) {
	retriever := RetrieverBackend(noopRetrieverBackend{})
	syncBackend := SyncBackend(noopSyncBackend{})

	switch resolveMemoryProviderName(cfg.Memory) {
	case "openviking":
		retriever = openvikingmemory.NewRetriever(cfg.OpenViking)
		syncBackend = openvikingmemory.NewSyncBackend(cfg.OpenViking)
	case "mem0":
		log.Printf("memory provider mem0 is not implemented yet, long-term memory disabled")
	default:
		log.Printf("memory provider %q is not supported, long-term memory disabled", cfg.Memory.Provider)
	}

	memoryProviderMu.Lock()
	activeRetriever = retriever
	activeSyncBackend = syncBackend
	memoryProviderMu.Unlock()

	applyMemorySyncBackendConfig(syncBackend)
}

// currentRetrieverBackend returns the active retrieval backend.
func currentRetrieverBackend() RetrieverBackend {
	memoryProviderMu.RLock()
	defer memoryProviderMu.RUnlock()
	return activeRetriever
}

// currentSyncBackend returns the active synchronization backend.
func currentSyncBackend() SyncBackend {
	memoryProviderMu.RLock()
	defer memoryProviderMu.RUnlock()
	return activeSyncBackend
}

// memorySyncConfigured reports whether the active provider supports persistence sync.
func memorySyncConfigured() bool {
	return currentSyncBackend().Configured()
}

// resolveMemoryProviderName keeps OpenViking as the backward-compatible default provider.
func resolveMemoryProviderName(cfg config.MemoryProviderConfig) string {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if provider == "" {
		return "openviking"
	}
	return provider
}
