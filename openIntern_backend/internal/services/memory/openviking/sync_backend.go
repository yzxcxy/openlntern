package openviking

import (
	"context"
	"fmt"
	"strings"
	"time"

	"openIntern/internal/config"
	"openIntern/internal/dao"
	"openIntern/internal/models"
	"openIntern/internal/services/memory/contracts"
)

const (
	defaultMemorySyncDelay      = 5 * time.Minute
	defaultMemorySyncPoll       = 3 * time.Second
	defaultMemorySyncTimeout    = 10 * time.Minute
	defaultMemorySyncRetryDelay = 30 * time.Second
)

// SyncBackend implements OpenViking-backed memory session synchronization.
type SyncBackend struct {
	syncDelay      time.Duration
	syncPoll       time.Duration
	syncTimeout    time.Duration
	syncRetryDelay time.Duration
}

// NewSyncBackend builds one OpenViking synchronization backend from configuration.
func NewSyncBackend(cfg config.OpenVikingConfig) *SyncBackend {
	return &SyncBackend{
		syncDelay:      durationFromSeconds(cfg.MemorySyncDelaySeconds, defaultMemorySyncDelay),
		syncPoll:       durationFromSeconds(cfg.MemorySyncPollSeconds, defaultMemorySyncPoll),
		syncTimeout:    durationFromSeconds(cfg.MemorySyncTimeoutSeconds, defaultMemorySyncTimeout),
		syncRetryDelay: durationFromSeconds(cfg.MemorySyncRetrySeconds, defaultMemorySyncRetryDelay),
	}
}

// Configured reports whether OpenViking session APIs are available.
func (b *SyncBackend) Configured() bool {
	return dao.OpenVikingSession.Configured()
}

// SyncDelay returns the first sync-attempt delay after one run completes.
func (b *SyncBackend) SyncDelay() time.Duration {
	return b.syncDelay
}

// SyncPollInterval returns the async commit task polling interval.
func (b *SyncBackend) SyncPollInterval() time.Duration {
	return b.syncPoll
}

// SyncTimeout returns the single-thread synchronization timeout.
func (b *SyncBackend) SyncTimeout() time.Duration {
	return b.syncTimeout
}

// SyncRetryDelay returns the retry interval after one sync failure.
func (b *SyncBackend) SyncRetryDelay() time.Duration {
	return b.syncRetryDelay
}

// SubmitMessages appends the provided messages and submits one async OpenViking commit operation.
func (b *SyncBackend) SubmitMessages(ctx context.Context, state models.MemorySyncState, messages []contracts.SyncMessage) (string, error) {
	sessionID, err := b.resolveSessionID(state)
	if err != nil {
		return "", err
	}
	for _, item := range messages {
		if err := dao.OpenVikingSession.AddMessage(ctx, sessionID, item.Role, item.Content); err != nil {
			return "", err
		}
	}
	result, err := dao.OpenVikingSession.Commit(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if result == nil {
		return "", nil
	}
	return strings.TrimSpace(result.TaskID), nil
}

// PollOperation reads one async OpenViking commit operation snapshot.
func (b *SyncBackend) PollOperation(ctx context.Context, operationID string) (string, string, error) {
	result, err := dao.OpenVikingSession.GetTask(ctx, operationID)
	if err != nil {
		return "", "", err
	}
	if result == nil {
		return "", "", nil
	}
	return strings.TrimSpace(result.Status), strings.TrimSpace(result.Error), nil
}

// resolveSessionID follows the thread-scoped deterministic session id rule used by OpenViking.
func (b *SyncBackend) resolveSessionID(state models.MemorySyncState) (string, error) {
	threadID := strings.TrimSpace(state.ThreadID)
	if threadID == "" {
		return "", fmt.Errorf("thread_id is required")
	}
	return threadID, nil
}
