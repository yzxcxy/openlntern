package memory

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

// ResetLegacySyncing converts legacy pre-Redis syncing rows back to pending so the new worker can resume them.
func (s *MemorySyncStateService) ResetLegacySyncing() error {
	return dao.MemorySyncState.ResetLegacySyncing()
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
	nextAttemptAt := nextMemorySyncScheduledAt(time.Now())
	if existing, err := dao.MemorySyncState.GetByThreadID(threadID); err == nil {
		// Keep fast polling cadence when an async commit task is already in-flight.
		if strings.TrimSpace(existing.CommitTaskID) != "" {
			nextAttemptAt = nextMemorySyncPollAt(time.Now())
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	item := &models.MemorySyncState{
		ThreadID:           threadID,
		LastCommittedRunID: runID,
		CommitStatus:       models.MemoryCommitStatusPending,
		Status:             models.MemorySyncStatusPending,
		RetryCount:         0,
		LastError:          "",
		NextAttemptAt:      nextAttemptAt,
	}
	return dao.MemorySyncState.UpsertPendingRun(item)
}

// MarkReady marks the thread as fully synchronized and updates the cursor.
func (s *MemorySyncStateService) MarkReady(threadID, lastSyncedMsgID, runID string) error {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return errors.New("thread_id is required")
	}
	return dao.MemorySyncState.MarkReady(
		threadID,
		strings.TrimSpace(lastSyncedMsgID),
		strings.TrimSpace(runID),
	)
}

// MarkFailed stores the latest synchronization error for the thread and schedules the next retry time.
func (s *MemorySyncStateService) MarkFailed(threadID, lastError string, nextAttemptAt *time.Time, commitStatus string) error {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return errors.New("thread_id is required")
	}
	commitStatus = strings.TrimSpace(commitStatus)
	if commitStatus == "" {
		commitStatus = models.MemoryCommitStatusPending
	}
	return dao.MemorySyncState.MarkFailed(
		threadID,
		strings.TrimSpace(lastError),
		nextAttemptAt,
		commitStatus,
	)
}

// MarkCommitSubmitted stores the accepted async commit task id and schedules next polling.
func (s *MemorySyncStateService) MarkCommitSubmitted(threadID, taskID string, nextAttemptAt *time.Time) error {
	threadID = strings.TrimSpace(threadID)
	taskID = strings.TrimSpace(taskID)
	if threadID == "" {
		return errors.New("thread_id is required")
	}
	if taskID == "" {
		return errors.New("task_id is required")
	}
	return dao.MemorySyncState.MarkCommitSubmitted(threadID, taskID, nextAttemptAt)
}

// MarkCommitPolling records the latest async task status and re-queues polling.
func (s *MemorySyncStateService) MarkCommitPolling(threadID, taskStatus string, nextAttemptAt *time.Time) error {
	threadID = strings.TrimSpace(threadID)
	taskStatus = strings.TrimSpace(taskStatus)
	if threadID == "" {
		return errors.New("thread_id is required")
	}
	if taskStatus == "" {
		return errors.New("task_status is required")
	}
	return dao.MemorySyncState.MarkCommitPolling(threadID, taskStatus, nextAttemptAt)
}

// ClearCommitTask clears stale async commit task metadata.
func (s *MemorySyncStateService) ClearCommitTask(threadID string) error {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return errors.New("thread_id is required")
	}
	return dao.MemorySyncState.ClearCommitTask(threadID)
}

// MarkMessagesAdded records that add-message phase has completed up to the provided cursor.
func (s *MemorySyncStateService) MarkMessagesAdded(threadID, lastAddedMsgID string) error {
	threadID = strings.TrimSpace(threadID)
	lastAddedMsgID = strings.TrimSpace(lastAddedMsgID)
	if threadID == "" {
		return errors.New("thread_id is required")
	}
	if lastAddedMsgID == "" {
		return errors.New("last_added_msg_id is required")
	}
	return dao.MemorySyncState.MarkMessagesAdded(threadID, lastAddedMsgID)
}

// IsNotFound reports whether the error means the sync state row does not exist.
func (s *MemorySyncStateService) IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
