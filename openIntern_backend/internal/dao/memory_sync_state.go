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

// UpsertPendingRun records that a completed chat run needs long-term memory synchronization.
func (d *MemorySyncStateDAO) UpsertPendingRun(item *models.MemorySyncState) error {
	return database.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "thread_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"last_committed_run_id",
			"status",
			"retry_count",
			"last_error",
			"next_attempt_at",
		}),
	}).Create(item).Error
}

// MarkSyncing transitions the state into syncing when it is runnable.
func (d *MemorySyncStateDAO) MarkSyncing(threadID string) (bool, error) {
	result := database.DB.Model(&models.MemorySyncState{}).
		Where("thread_id = ?", threadID).
		Where("status IN ?", []string{models.MemorySyncStatusPending, models.MemorySyncStatusFailed}).
		Where("next_attempt_at IS NULL OR next_attempt_at <= ?", time.Now()).
		Updates(map[string]any{
			"status": models.MemorySyncStatusSyncing,
		})
	return result.RowsAffected > 0, result.Error
}

// UpdateSessionID stores the OpenViking session id as soon as it is known.
func (d *MemorySyncStateDAO) UpdateSessionID(threadID, sessionID string) error {
	return database.DB.Model(&models.MemorySyncState{}).
		Where("thread_id = ?", threadID).
		Update("openviking_session_id", sessionID).Error
}

// MarkReady records the latest synchronized cursor and clears failure metadata.
func (d *MemorySyncStateDAO) MarkReady(threadID, sessionID, lastSyncedMsgID, runID string) error {
	return database.DB.Model(&models.MemorySyncState{}).
		Where("thread_id = ?", threadID).
		Updates(map[string]any{
			"openviking_session_id": sessionID,
			"last_synced_msg_id":    lastSyncedMsgID,
			"last_committed_run_id": runID,
			"status":                models.MemorySyncStatusReady,
			"retry_count":           0,
			"last_error":            "",
			"next_attempt_at":       nil,
		}).Error
}

// MarkFailed stores the latest failure and increments retry counters.
func (d *MemorySyncStateDAO) MarkFailed(threadID, lastError string, nextAttemptAt *time.Time) error {
	return database.DB.Model(&models.MemorySyncState{}).
		Where("thread_id = ?", threadID).
		Updates(map[string]any{
			"status":          models.MemorySyncStatusFailed,
			"retry_count":     gorm.Expr("retry_count + 1"),
			"last_error":      lastError,
			"next_attempt_at": nextAttemptAt,
		}).Error
}
