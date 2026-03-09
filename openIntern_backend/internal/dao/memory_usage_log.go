package dao

import (
	"time"

	"openIntern/internal/database"
	"openIntern/internal/models"

	"gorm.io/gorm/clause"
)

// MemoryUsageLogDAO provides persistence methods for per-run memory usage records.
type MemoryUsageLogDAO struct{}

// MemoryUsageLog is the shared DAO singleton for memory usage tracking.
var MemoryUsageLog = new(MemoryUsageLogDAO)

// CreateBatch inserts one batch of memory usage rows and ignores duplicate run+uri pairs.
func (d *MemoryUsageLogDAO) CreateBatch(items []models.MemoryUsageLog) error {
	if len(items) == 0 {
		return nil
	}
	return database.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "thread_id"},
			{Name: "run_id"},
			{Name: "memory_uri"},
		},
		DoNothing: true,
	}).Create(&items).Error
}

// ListPendingByThreadID loads all usage rows that have not yet been reported to OpenViking.
func (d *MemoryUsageLogDAO) ListPendingByThreadID(threadID string) ([]models.MemoryUsageLog, error) {
	var items []models.MemoryUsageLog
	if err := database.DB.
		Where("thread_id = ?", threadID).
		Where("reported_at IS NULL").
		Order("id ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// MarkReported sets reported_at for the specified rows after OpenViking accepts the usage records.
func (d *MemoryUsageLogDAO) MarkReported(ids []uint, reportedAt time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	return database.DB.Model(&models.MemoryUsageLog{}).
		Where("id IN ?", ids).
		Where("reported_at IS NULL").
		Update("reported_at", reportedAt).Error
}
