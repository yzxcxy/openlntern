package services

import (
	"errors"
	"strings"
	"time"

	"openIntern/internal/dao"
	"openIntern/internal/models"

	"gorm.io/gorm"
)

// MemorySyncStateService manages thread-level long-term memory sync state.
type MemorySyncStateService struct{}

// MemorySyncState is the shared service singleton for memory sync state access.
var MemorySyncState = new(MemorySyncStateService)

// GetByThreadID returns the memory sync state row for the specified thread.
func (s *MemorySyncStateService) GetByThreadID(threadID string) (*models.MemorySyncState, error) {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return nil, errors.New("thread_id is required")
	}
	return dao.MemorySyncState.GetByThreadID(threadID)
}

// ListRunnable returns a bounded batch of pending or failed sync states.
func (s *MemorySyncStateService) ListRunnable(limit int) ([]models.MemorySyncState, error) {
	return dao.MemorySyncState.ListRunnable(limit)
}

// ScheduleThreadSync marks the thread as pending long-term memory synchronization.
func (s *MemorySyncStateService) ScheduleThreadSync(threadID, runID string) error {
	if !dao.OpenVikingSession.Configured() {
		return nil
	}
	threadID = strings.TrimSpace(threadID)
	runID = strings.TrimSpace(runID)
	if threadID == "" {
		return errors.New("thread_id is required")
	}
	if runID == "" {
		return errors.New("run_id is required")
	}
	if _, err := dao.Thread.GetByThreadID(threadID); err != nil {
		return err
	}
	item := &models.MemorySyncState{
		ThreadID:           threadID,
		LastCommittedRunID: runID,
		Status:             models.MemorySyncStatusPending,
		RetryCount:         0,
		LastError:          "",
		NextAttemptAt:      nextMemorySyncScheduledAt(time.Now()),
	}
	return dao.MemorySyncState.UpsertPendingRun(item)
}

// MarkSyncing transitions the specified thread to syncing when it is runnable.
func (s *MemorySyncStateService) MarkSyncing(threadID string) (bool, error) {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return false, errors.New("thread_id is required")
	}
	return dao.MemorySyncState.MarkSyncing(threadID)
}

// UpdateSessionID stores the OpenViking session id as soon as the session creation call succeeds.
func (s *MemorySyncStateService) UpdateSessionID(threadID, sessionID string) error {
	threadID = strings.TrimSpace(threadID)
	sessionID = strings.TrimSpace(sessionID)
	if threadID == "" {
		return errors.New("thread_id is required")
	}
	if sessionID == "" {
		return errors.New("session_id is required")
	}
	return dao.MemorySyncState.UpdateSessionID(threadID, sessionID)
}

// MarkReady marks the thread as fully synchronized and updates the cursor.
func (s *MemorySyncStateService) MarkReady(threadID, sessionID, lastSyncedMsgID, runID string) error {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return errors.New("thread_id is required")
	}
	return dao.MemorySyncState.MarkReady(
		threadID,
		strings.TrimSpace(sessionID),
		strings.TrimSpace(lastSyncedMsgID),
		strings.TrimSpace(runID),
	)
}

// MarkFailed stores the latest synchronization error for the thread and schedules the next retry time.
func (s *MemorySyncStateService) MarkFailed(threadID, lastError string, nextAttemptAt *time.Time) error {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return errors.New("thread_id is required")
	}
	return dao.MemorySyncState.MarkFailed(threadID, strings.TrimSpace(lastError), nextAttemptAt)
}

// IsNotFound reports whether the error means the sync state row does not exist.
func (s *MemorySyncStateService) IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
