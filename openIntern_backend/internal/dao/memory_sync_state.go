package dao

import (
	"openIntern/internal/database"
	"openIntern/internal/models"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MemorySyncStateDAO provides persistence methods for thread memory sync state.
type MemorySyncStateDAO struct{}

// MemorySyncState is the shared DAO singleton.
var MemorySyncState = new(MemorySyncStateDAO)

// GetByThreadID loads the sync state row for the specified thread.
func (d *MemorySyncStateDAO) GetByThreadID(threadID string) (*models.MemorySyncState, error) {
	var item models.MemorySyncState
	if err := database.DB.Where("thread_id = ?", threadID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *MemorySyncStateDAO) GetByUserIDAndThreadID(userID, threadID string) (*models.MemorySyncState, error) {
	var item models.MemorySyncState
	if err := database.DB.Where("user_id = ? AND thread_id = ?", userID, threadID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// ListRunnable returns a bounded batch of states that are ready for synchronization work.
func (d *MemorySyncStateDAO) ListRunnable(limit int) ([]models.MemorySyncState, error) {
	if limit <= 0 {
		limit = 20
	}
	var items []models.MemorySyncState
	now := time.Now()
	if err := database.DB.
		Where("status IN ?", []string{models.MemorySyncStatusPending, models.MemorySyncStatusFailed}).
		Where("next_attempt_at IS NULL OR next_attempt_at <= ?", now).
		Order("updated_at ASC").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ResetLegacySyncing converts old syncing states into pending so Redis-based workers can resume them.
func (d *MemorySyncStateDAO) ResetLegacySyncing() error {
	return database.DB.Model(&models.MemorySyncState{}).
		Where("status = ?", "syncing").
		Updates(map[string]any{
			"status": models.MemorySyncStatusPending,
		}).Error
}

// UpsertPendingRun records that a completed chat run needs long-term memory synchronization.
func (d *MemorySyncStateDAO) UpsertPendingRun(item *models.MemorySyncState) error {
	return database.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "thread_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"last_committed_run_id",
			"status",
			"retry_count",
			"last_error",
			"next_attempt_at",
		}),
	}).Create(item).Error
}

// MarkReady records the latest synchronized cursor and clears failure metadata.
func (d *MemorySyncStateDAO) MarkReady(threadID, lastSyncedMsgID, runID string) error {
	return database.DB.Model(&models.MemorySyncState{}).
		Where("thread_id = ?", threadID).
		Updates(map[string]any{
			"last_added_msg_id":     lastSyncedMsgID,
			"last_synced_msg_id":    lastSyncedMsgID,
			"last_committed_run_id": runID,
			"commit_task_id":        "",
			"commit_task_status":    "",
			"commit_status":         models.MemoryCommitStatusCommitted,
			"status":                models.MemorySyncStatusReady,
			"retry_count":           0,
			"last_error":            "",
			"next_attempt_at":       nil,
		}).Error
}

func (d *MemorySyncStateDAO) MarkReadyByUserID(userID, threadID, lastSyncedMsgID, runID string) error {
	return database.DB.Model(&models.MemorySyncState{}).
		Where("user_id = ? AND thread_id = ?", userID, threadID).
		Updates(map[string]any{
			"last_added_msg_id":     lastSyncedMsgID,
			"last_synced_msg_id":    lastSyncedMsgID,
			"last_committed_run_id": runID,
			"commit_task_id":        "",
			"commit_task_status":    "",
			"commit_status":         models.MemoryCommitStatusCommitted,
			"status":                models.MemorySyncStatusReady,
			"retry_count":           0,
			"last_error":            "",
			"next_attempt_at":       nil,
		}).Error
}

// MarkFailed stores the latest failure and increments retry counters.
func (d *MemorySyncStateDAO) MarkFailed(threadID, lastError string, nextAttemptAt *time.Time, commitStatus string) error {
	return database.DB.Model(&models.MemorySyncState{}).
		Where("thread_id = ?", threadID).
		Updates(map[string]any{
			"commit_status":   commitStatus,
			"status":          models.MemorySyncStatusFailed,
			"retry_count":     gorm.Expr("retry_count + 1"),
			"last_error":      lastError,
			"next_attempt_at": nextAttemptAt,
		}).Error
}

func (d *MemorySyncStateDAO) MarkFailedByUserID(userID, threadID, lastError string, nextAttemptAt *time.Time, commitStatus string) error {
	return database.DB.Model(&models.MemorySyncState{}).
		Where("user_id = ? AND thread_id = ?", userID, threadID).
		Updates(map[string]any{
			"commit_status":   commitStatus,
			"status":          models.MemorySyncStatusFailed,
			"retry_count":     gorm.Expr("retry_count + 1"),
			"last_error":      lastError,
			"next_attempt_at": nextAttemptAt,
		}).Error
}

// MarkCommitSubmitted records the accepted async task id after commit(wait=false) submission.
func (d *MemorySyncStateDAO) MarkCommitSubmitted(threadID, taskID string, nextAttemptAt *time.Time) error {
	return database.DB.Model(&models.MemorySyncState{}).
		Where("thread_id = ?", threadID).
		Updates(map[string]any{
			"commit_task_id":     taskID,
			"commit_task_status": models.MemorySyncStatusPending,
			"commit_status":      models.MemoryCommitStatusProcessing,
			"status":             models.MemorySyncStatusPending,
			"last_error":         "",
			"next_attempt_at":    nextAttemptAt,
		}).Error
}

func (d *MemorySyncStateDAO) MarkCommitSubmittedByUserID(userID, threadID, taskID string, nextAttemptAt *time.Time) error {
	return database.DB.Model(&models.MemorySyncState{}).
		Where("user_id = ? AND thread_id = ?", userID, threadID).
		Updates(map[string]any{
			"commit_task_id":     taskID,
			"commit_task_status": models.MemorySyncStatusPending,
			"commit_status":      models.MemoryCommitStatusProcessing,
			"status":             models.MemorySyncStatusPending,
			"last_error":         "",
			"next_attempt_at":    nextAttemptAt,
		}).Error
}

// MarkCommitPolling stores the latest polled task status and re-queues the thread for next poll.
func (d *MemorySyncStateDAO) MarkCommitPolling(threadID, taskStatus string, nextAttemptAt *time.Time) error {
	return database.DB.Model(&models.MemorySyncState{}).
		Where("thread_id = ?", threadID).
		Updates(map[string]any{
			"commit_task_status": taskStatus,
			"commit_status":      models.MemoryCommitStatusProcessing,
			"status":             models.MemorySyncStatusPending,
			"last_error":         "",
			"next_attempt_at":    nextAttemptAt,
		}).Error
}

func (d *MemorySyncStateDAO) MarkCommitPollingByUserID(userID, threadID, taskStatus string, nextAttemptAt *time.Time) error {
	return database.DB.Model(&models.MemorySyncState{}).
		Where("user_id = ? AND thread_id = ?", userID, threadID).
		Updates(map[string]any{
			"commit_task_status": taskStatus,
			"commit_status":      models.MemoryCommitStatusProcessing,
			"status":             models.MemorySyncStatusPending,
			"last_error":         "",
			"next_attempt_at":    nextAttemptAt,
		}).Error
}

// ClearCommitTask removes stale async task metadata so commit can be re-submitted safely.
func (d *MemorySyncStateDAO) ClearCommitTask(threadID string) error {
	return database.DB.Model(&models.MemorySyncState{}).
		Where("thread_id = ?", threadID).
		Updates(map[string]any{
			"commit_task_id":     "",
			"commit_task_status": "",
		}).Error
}

func (d *MemorySyncStateDAO) ClearCommitTaskByUserID(userID, threadID string) error {
	return database.DB.Model(&models.MemorySyncState{}).
		Where("user_id = ? AND thread_id = ?", userID, threadID).
		Updates(map[string]any{
			"commit_task_id":     "",
			"commit_task_status": "",
		}).Error
}

// MarkMessagesAdded stores the add-message cursor before commit starts.
func (d *MemorySyncStateDAO) MarkMessagesAdded(threadID, lastAddedMsgID string) error {
	return database.DB.Model(&models.MemorySyncState{}).
		Where("thread_id = ?", threadID).
		Updates(map[string]any{
			"last_added_msg_id":  lastAddedMsgID,
			"commit_task_id":     "",
			"commit_task_status": "",
			"commit_status":      models.MemoryCommitStatusPending,
			"last_error":         "",
		}).Error
}

func (d *MemorySyncStateDAO) MarkMessagesAddedByUserID(userID, threadID, lastAddedMsgID string) error {
	return database.DB.Model(&models.MemorySyncState{}).
		Where("user_id = ? AND thread_id = ?", userID, threadID).
		Updates(map[string]any{
			"last_added_msg_id":  lastAddedMsgID,
			"commit_task_id":     "",
			"commit_task_status": "",
			"commit_status":      models.MemoryCommitStatusPending,
			"last_error":         "",
		}).Error
}
