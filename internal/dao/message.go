package dao

import (
	"openIntern/internal/database"
	"openIntern/internal/models"
)

type MessageDAO struct{}

var Message = new(MessageDAO)

func (d *MessageDAO) ListByThreadID(threadID string, page, pageSize int) ([]models.Message, int64, error) {
	_, pageSize, offset := normalizePagination(page, pageSize, 20)

	query := database.DB.Model(&models.Message{}).Where("thread_id = ?", threadID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []models.Message
	if err := query.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (d *MessageDAO) ListAllByThreadID(threadID string) ([]models.Message, error) {
	var items []models.Message
	if err := database.DB.Where("thread_id = ?", threadID).Order("created_at asc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (d *MessageDAO) CreateBatch(messages []models.Message) error {
	return database.DB.Create(&messages).Error
}
