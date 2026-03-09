package services

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
	"gorm.io/gorm"
)

type ThreadService struct{}

var Thread = new(ThreadService)

func (s *ThreadService) ListThreads(page, pageSize int) ([]models.Thread, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	key := fmt.Sprintf("threads:%d:%d", page, pageSize)
	ctx := context.Background()
	client := database.GetRedis()
	if client != nil {
		if cached, err := client.Get(ctx, key).Result(); err == nil {
			var payload struct {
				Items []models.Thread `json:"items"`
				Total int64           `json:"total"`
			}
			if err := json.Unmarshal([]byte(cached), &payload); err == nil {
				return payload.Items, payload.Total, nil
			}
		}
	}

	threads, total, err := dao.Thread.List(page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	if client != nil {
		payload := struct {
			Items []models.Thread `json:"items"`
			Total int64           `json:"total"`
		}{
			Items: threads,
			Total: total,
		}
		if b, err := json.Marshal(payload); err == nil {
			client.Set(ctx, key, b, 60*time.Second)
		}
	}

	return threads, total, nil
}

func (s *ThreadService) GetThread(threadID string) (*models.Thread, error) {
	if threadID == "" {
		return nil, errors.New("thread_id is required")
	}
	key := fmt.Sprintf("thread:%s", threadID)
	ctx := context.Background()
	client := database.GetRedis()
	if client != nil {
		if cached, err := client.Get(ctx, key).Result(); err == nil {
			var thread models.Thread
			if err := json.Unmarshal([]byte(cached), &thread); err == nil {
				return &thread, nil
			}
		} else if err != redis.Nil {
			return nil, err
		}
	}

	thread, err := dao.Thread.GetByThreadID(threadID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}

	if client != nil {
		if b, err := json.Marshal(thread); err == nil {
			client.Set(ctx, key, b, 60*time.Second)
		}
	}

	return thread, nil
}

func (s *ThreadService) GetThreadByThreadID(threadID string) (*models.Thread, error) {
	if threadID == "" {
		return nil, errors.New("thread_id is required")
	}
	return dao.Thread.GetByThreadID(threadID)
}

func (s *ThreadService) EnsureThread(threadID, title string) (*models.Thread, error) {
	if threadID == "" {
		return nil, errors.New("thread_id is required")
	}
	thread, err := dao.Thread.GetByThreadID(threadID)
	if err == nil {
		updates := map[string]any{}
		if thread.Title == "" && title != "" {
			updates["title"] = title
			thread.Title = title
		}
		if len(updates) > 0 {
			if err := dao.Thread.UpdateFields(thread, updates); err != nil {
				return nil, err
			}
			invalidateThreadCache(thread.ThreadID)
		}
		return thread, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		thread = &models.Thread{
			ThreadID: threadID,
			Title:    title,
		}
		if err := dao.Thread.Create(thread); err != nil {
			return nil, err
		}
		invalidateThreadCache(thread.ThreadID)
		return thread, nil
	}
	return nil, err
}

func (s *ThreadService) UpdateThreadTitle(threadID, title string) error {
	if threadID == "" {
		return errors.New("thread_id is required")
	}
	if title == "" {
		return errors.New("title is required")
	}
	if _, err := dao.Thread.GetByThreadID(threadID); err != nil {
		return err
	}
	if _, err := dao.Thread.UpdateTitle(threadID, title); err != nil {
		return err
	}
	invalidateThreadCache(threadID)
	return nil
}

func (s *ThreadService) DeleteThread(threadID string) error {
	if threadID == "" {
		return errors.New("thread_id is required")
	}
	if err := dao.Thread.DeleteWithMessages(threadID); err != nil {
		return err
	}
	invalidateThreadCache(threadID)
	return nil
}

func (s *ThreadService) TouchThread(threadID string) error {
	if threadID == "" {
		return errors.New("thread_id is required")
	}
	if _, err := dao.Thread.GetByThreadID(threadID); err != nil {
		return err
	}
	if _, err := dao.Thread.Touch(threadID, time.Now()); err != nil {
		return err
	}
	invalidateThreadCache(threadID)
	return nil
}

func invalidateThreadCache(threadID string) {
	client := database.GetRedis()
	if client == nil {
		return
	}
	ctx := context.Background()
	if threadID != "" {
		client.Del(ctx, fmt.Sprintf("thread:%s", threadID))
	}
	pattern := "threads:*"
	var cursor uint64
	for {
		keys, next, err := client.Scan(ctx, cursor, pattern, 50).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			client.Del(ctx, keys...)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
}
