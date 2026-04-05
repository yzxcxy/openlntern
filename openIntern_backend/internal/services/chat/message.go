package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"openIntern/internal/dao"
	"openIntern/internal/database"
	"openIntern/internal/models"

	"github.com/redis/go-redis/v9"
)

type MessageService struct{}

var Message = new(MessageService)

func (s *MessageService) ListMessages(userID, threadID string, page, pageSize int) ([]models.Message, int64, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, 0, errors.New("user_id is required")
	}
	if threadID == "" {
		return nil, 0, errors.New("thread_id is required")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	if _, err := dao.Thread.GetByUserIDAndThreadID(userID, threadID); err != nil {
		return nil, 0, err
	}

	key := fmt.Sprintf("messages:%s:%s:%d:%d", userID, threadID, page, pageSize)
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

	messages, total, err := dao.Message.ListByUserIDAndThreadID(userID, threadID, page, pageSize)
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

func (s *MessageService) ListThreadMessages(userID, threadID string) ([]models.Message, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("user_id is required")
	}
	if threadID == "" {
		return nil, errors.New("thread_id is required")
	}
	return dao.Message.ListAllByUserIDAndThreadID(userID, threadID)
}

func (s *MessageService) CreateMessages(messages []models.Message) error {
	if len(messages) == 0 {
		return nil
	}
	return dao.Message.CreateBatch(messages)
}
