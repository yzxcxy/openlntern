package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"openIntern/internal/database"
	"openIntern/internal/models"

	"github.com/redis/go-redis/v9"
)

type MessageService struct{}

var Message = new(MessageService)

func (s *MessageService) ListMessages(ownerID, threadID string, page, pageSize int) ([]models.Message, int64, error) {
	if ownerID == "" || threadID == "" {
		return nil, 0, errors.New("owner_id and thread_id are required")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	var thread models.Thread
	if err := database.DB.Where("owner_id = ? AND thread_id = ?", ownerID, threadID).First(&thread).Error; err != nil {
		return nil, 0, err
	}

	key := fmt.Sprintf("messages:%s:%d:%d", threadID, page, pageSize)
	ctx := context.Background()
	client := database.GetRedis()
	if client != nil {
		if cached, err := client.Get(ctx, key).Result(); err == nil {
			var payload struct {
				Items []models.Message `json:"items"`
				Total int64            `json:"total"`
			}
			if err := json.Unmarshal([]byte(cached), &payload); err == nil {
				return payload.Items, payload.Total, nil
			}
		} else if err != redis.Nil {
			return nil, 0, err
		}
	}

	var messages []models.Message
	var total int64
	db := database.DB.Model(&models.Message{}).Where("thread_id = ?", threadID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := db.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&messages).Error; err != nil {
		return nil, 0, err
	}

	if client != nil {
		payload := struct {
			Items []models.Message `json:"items"`
			Total int64            `json:"total"`
		}{
			Items: messages,
			Total: total,
		}
		if b, err := json.Marshal(payload); err == nil {
			client.Set(ctx, key, b, 60*time.Second)
		}
	}

	return messages, total, nil
}
