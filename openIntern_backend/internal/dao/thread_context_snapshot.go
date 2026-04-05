package dao

import (
	"openIntern/internal/database"
	"openIntern/internal/models"
)

// ThreadContextSnapshotDAO provides persistence methods for thread compression snapshots.
type ThreadContextSnapshotDAO struct{}

// ThreadContextSnapshot is the shared DAO singleton.
var ThreadContextSnapshot = new(ThreadContextSnapshotDAO)

// Create inserts a thread context snapshot record.
func (d *ThreadContextSnapshotDAO) Create(item *models.ThreadContextSnapshot) error {
	return database.DB.Create(item).Error
}

// GetLatestByThreadID returns the latest snapshot ordered by compression index.
func (d *ThreadContextSnapshotDAO) GetLatestByThreadID(threadID string) (*models.ThreadContextSnapshot, error) {
	var item models.ThreadContextSnapshot
	result := database.DB.
		Where("thread_id = ?", threadID).
		Order("compression_index DESC").
		Order("created_at DESC").
		Limit(1).
		Find(&item)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &item, nil
}

func (d *ThreadContextSnapshotDAO) GetLatestByUserIDAndThreadID(userID, threadID string) (*models.ThreadContextSnapshot, error) {
	var item models.ThreadContextSnapshot
	result := database.DB.
		Where("user_id = ? AND thread_id = ?", userID, threadID).
		Order("compression_index DESC").
		Order("created_at DESC").
		Limit(1).
		Find(&item)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &item, nil
}
