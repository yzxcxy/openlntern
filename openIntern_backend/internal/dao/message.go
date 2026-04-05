package dao

import (
	"openIntern/internal/database"
	"openIntern/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	if err := query.Order("created_at desc, id desc").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (d *MessageDAO) ListByUserIDAndThreadID(userID, threadID string, page, pageSize int) ([]models.Message, int64, error) {
	_, pageSize, offset := normalizePagination(page, pageSize, 20)

	query := database.DB.Model(&models.Message{}).Where("user_id = ? AND thread_id = ?", userID, threadID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []models.Message
	if err := query.Order("created_at desc, id desc").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (d *MessageDAO) ListAllByThreadID(threadID string) ([]models.Message, error) {
	var items []models.Message
	if err := database.DB.Where("thread_id = ?", threadID).Order("created_at asc, id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (d *MessageDAO) ListAllByUserIDAndThreadID(userID, threadID string) ([]models.Message, error) {
	var items []models.Message
	if err := database.DB.Where("user_id = ? AND thread_id = ?", userID, threadID).Order("created_at asc, id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (d *MessageDAO) CreateBatch(messages []models.Message) error {
	if len(messages) == 0 {
		return nil
	}

	return database.DB.Transaction(func(tx *gorm.DB) error {
		grouped := make(map[string][]int, len(messages))
		threadOrder := make([]string, 0, len(messages))
		for index := range messages {
			userThreadKey := messages[index].UserID + ":" + messages[index].ThreadID
			if _, exists := grouped[userThreadKey]; !exists {
				threadOrder = append(threadOrder, userThreadKey)
			}
			grouped[userThreadKey] = append(grouped[userThreadKey], index)
		}

		for _, userThreadKey := range threadOrder {
			if userThreadKey == ":" {
				continue
			}
			messageIndexes := grouped[userThreadKey]
			first := messages[messageIndexes[0]]
			userID := first.UserID
			threadID := first.ThreadID

			// 锁住 thread 行，保证同一 thread 下分配 sequence 时不会并发交叉。
			var thread models.Thread
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("user_id = ? AND thread_id = ?", userID, threadID).
				First(&thread).Error; err != nil {
				return err
			}

			var latest models.Message
			nextSequence := int64(1)
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("user_id = ? AND thread_id = ?", userID, threadID).
				Order("sequence desc, id desc").
				Take(&latest).Error
			if err != nil {
				if err != gorm.ErrRecordNotFound {
					return err
				}
			} else {
				nextSequence = latest.Sequence + 1
			}

			for _, messageIndex := range messageIndexes {
				messages[messageIndex].Sequence = nextSequence
				nextSequence++
			}
		}

		return tx.Create(&messages).Error
	})
}
