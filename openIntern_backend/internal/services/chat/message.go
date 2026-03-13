package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"openIntern/internal/dao"
	"openIntern/internal/database"
	"openIntern/internal/models"

	"github.com/redis/go-redis/v9"
)

type MessageService struct{}

var Message = new(MessageService)

func (s *MessageService) ListMessages(threadID string, page, pageSize int) ([]models.Message, int64, error) {
	if threadID == "" {
		return nil, 0, errors.New("thread_id is required")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	if _, err := dao.Thread.GetByThreadID(threadID); err != nil {
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

	messages, total, err := dao.Message.ListByThreadID(threadID, page, pageSize)
	if err != nil {
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

func (s *MessageService) ListThreadMessages(threadID string) ([]models.Message, error) {
	if threadID == "" {
		return nil, errors.New("thread_id is required")
	}
	return dao.Message.ListAllByThreadID(threadID)
}

func (s *MessageService) CreateMessages(messages []models.Message) error {
	if len(messages) == 0 {
		return nil
	}
	return dao.Message.CreateBatch(messages)
}
