package dao

import (
	"time"

	"openIntern/internal/database"
	"openIntern/internal/models"

	"gorm.io/gorm"
)

type ThreadDAO struct{}

var Thread = new(ThreadDAO)

func (d *ThreadDAO) List(page, pageSize int) ([]models.Thread, int64, error) {
	_, pageSize, offset := normalizePagination(page, pageSize, 10)

	query := database.DB.Model(&models.Thread{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []models.Thread
	if err := query.Order("updated_at desc").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (d *ThreadDAO) ListByUserID(userID string, page, pageSize int) ([]models.Thread, int64, error) {
	_, pageSize, offset := normalizePagination(page, pageSize, 10)

	query := database.DB.Model(&models.Thread{}).Where("user_id = ?", userID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []models.Thread
	if err := query.Order("updated_at desc").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (d *ThreadDAO) GetByThreadID(threadID string) (*models.Thread, error) {
	var item models.Thread
	if err := database.DB.Where("thread_id = ?", threadID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *ThreadDAO) GetByUserIDAndThreadID(userID, threadID string) (*models.Thread, error) {
	var item models.Thread
	if err := database.DB.Where("user_id = ? AND thread_id = ?", userID, threadID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *ThreadDAO) Create(thread *models.Thread) error {
	return database.DB.Create(thread).Error
}

func (d *ThreadDAO) UpdateFields(thread *models.Thread, updates map[string]any) error {
	return database.DB.Model(thread).Updates(updates).Error
}

func (d *ThreadDAO) UpdateTitle(threadID, title string) (int64, error) {
	result := database.DB.Model(&models.Thread{}).Where("thread_id = ?", threadID).Update("title", title)
	return result.RowsAffected, result.Error
}

func (d *ThreadDAO) Touch(threadID string, updatedAt time.Time) (int64, error) {
	result := database.DB.Model(&models.Thread{}).Where("thread_id = ?", threadID).Update("updated_at", updatedAt)
	return result.RowsAffected, result.Error
}

func (d *ThreadDAO) UpdateTitleByUserID(userID, threadID, title string) (int64, error) {
	result := database.DB.Model(&models.Thread{}).
		Where("user_id = ? AND thread_id = ?", userID, threadID).
		Update("title", title)
	return result.RowsAffected, result.Error
}

func (d *ThreadDAO) TouchByUserID(userID, threadID string, updatedAt time.Time) (int64, error) {
	result := database.DB.Model(&models.Thread{}).
		Where("user_id = ? AND thread_id = ?", userID, threadID).
		Update("updated_at", updatedAt)
	return result.RowsAffected, result.Error
}

// DeleteWithMessages removes the thread and all thread-scoped persistence records in one transaction.
func (d *ThreadDAO) DeleteWithMessages(threadID string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var thread models.Thread
		if err := tx.Where("thread_id = ?", threadID).First(&thread).Error; err != nil {
			return err
		}
		if err := tx.Where("thread_id = ?", threadID).Delete(&models.Message{}).Error; err != nil {
			return err
		}
		if err := tx.Where("thread_id = ?", threadID).Delete(&models.MemorySyncState{}).Error; err != nil {
			return err
		}
		if err := tx.Where("thread_id = ?", threadID).Delete(&models.MemoryUsageLog{}).Error; err != nil {
			return err
		}
		return tx.Delete(&thread).Error
	})
}

func (d *ThreadDAO) DeleteWithMessagesByUserID(userID, threadID string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var thread models.Thread
		if err := tx.Where("user_id = ? AND thread_id = ?", userID, threadID).First(&thread).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ? AND thread_id = ?", userID, threadID).Delete(&models.Message{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ? AND thread_id = ?", userID, threadID).Delete(&models.MemorySyncState{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ? AND thread_id = ?", userID, threadID).Delete(&models.MemoryUsageLog{}).Error; err != nil {
			return err
		}
		return tx.Delete(&thread).Error
	})
}
