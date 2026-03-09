package dao

import (
	"errors"

	"openIntern/internal/database"
	"openIntern/internal/models"

	"gorm.io/gorm"
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
	err := database.DB.
		Where("thread_id = ?", threadID).
		Order("compression_index DESC").
		Order("created_at DESC").
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}
