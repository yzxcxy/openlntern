package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"openIntern/internal/database"
	"openIntern/internal/models"

	"github.com/redis/go-redis/v9"
)

// threadCache isolates thread Redis operations so the service can focus on business flow.
type threadCache struct{}

func (threadCache) getThreadList(page, pageSize int) ([]models.Thread, int64, bool, error) {
	client := database.GetRedis()
	if client == nil {
		return nil, 0, false, nil
	}

	key := fmt.Sprintf("threads:%d:%d", page, pageSize)
	cached, err := client.Get(context.Background(), key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, 0, false, nil
		}
		return nil, 0, false, err
	}

	var payload struct {
		Items []models.Thread `json:"items"`
		Total int64           `json:"total"`
	}
	if err := json.Unmarshal([]byte(cached), &payload); err != nil {
		return nil, 0, false, nil
	}

	return payload.Items, payload.Total, true, nil
}

func (threadCache) setThreadList(page, pageSize int, items []models.Thread, total int64) {
	client := database.GetRedis()
	if client == nil {
		return
	}

	payload := struct {
		Items []models.Thread `json:"items"`
		Total int64           `json:"total"`
	}{
		Items: items,
		Total: total,
	}
	if raw, err := json.Marshal(payload); err == nil {
		client.Set(context.Background(), fmt.Sprintf("threads:%d:%d", page, pageSize), raw, 60*time.Second)
	}
}

func (threadCache) getThread(threadID string) (*models.Thread, bool, error) {
	client := database.GetRedis()
	if client == nil {
		return nil, false, nil
	}

	cached, err := client.Get(context.Background(), fmt.Sprintf("thread:%s", threadID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}

	var thread models.Thread
	if err := json.Unmarshal([]byte(cached), &thread); err != nil {
		return nil, false, nil
	}
	return &thread, true, nil
}

func (threadCache) setThread(thread *models.Thread) {
	client := database.GetRedis()
	if client == nil || thread == nil {
		return
	}

	if raw, err := json.Marshal(thread); err == nil {
		client.Set(context.Background(), fmt.Sprintf("thread:%s", thread.ThreadID), raw, 60*time.Second)
	}
}

func (threadCache) invalidate(threadID string) {
	client := database.GetRedis()
	if client == nil {
		return
	}

	ctx := context.Background()
	if threadID != "" {
		client.Del(ctx, fmt.Sprintf("thread:%s", threadID))
	}

	var cursor uint64
	for {
		keys, next, err := client.Scan(ctx, cursor, "threads:*", 50).Result()
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

var threadListCache = threadCache{}
