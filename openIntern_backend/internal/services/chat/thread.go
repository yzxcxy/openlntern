package chat

import (
	"errors"
	"strings"
	"time"

	"openIntern/internal/dao"
	"openIntern/internal/models"

	"gorm.io/gorm"
)

type ThreadService struct{}

var Thread = new(ThreadService)

func (s *ThreadService) ListThreads(userID string, page, pageSize int) ([]models.Thread, int64, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, 0, errors.New("user_id is required")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if items, total, hit, err := threadListCache.getThreadList(userID, page, pageSize); err != nil {
		return nil, 0, err
	} else if hit {
		return items, total, nil
	}

	threads, total, err := dao.Thread.ListByUserID(userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	threadListCache.setThreadList(userID, page, pageSize, threads, total)

	return threads, total, nil
}

func (s *ThreadService) GetThread(userID, threadID string) (*models.Thread, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("user_id is required")
	}
	if threadID == "" {
		return nil, errors.New("thread_id is required")
	}
	if thread, hit, err := threadListCache.getThread(userID, threadID); err != nil {
		return nil, err
	} else if hit {
		return thread, nil
	}

	thread, err := dao.Thread.GetByUserIDAndThreadID(userID, threadID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}

	threadListCache.setThread(thread)

	return thread, nil
}

func (s *ThreadService) GetThreadByThreadID(userID, threadID string) (*models.Thread, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user_id is required")
	}
	if threadID == "" {
		return nil, errors.New("thread_id is required")
	}
	return dao.Thread.GetByUserIDAndThreadID(strings.TrimSpace(userID), threadID)
}

func (s *ThreadService) EnsureThread(userID, threadID, title string) (*models.Thread, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("user_id is required")
	}
	if threadID == "" {
		return nil, errors.New("thread_id is required")
	}
	thread, err := dao.Thread.GetByUserIDAndThreadID(userID, threadID)
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
			threadListCache.invalidate(userID, thread.ThreadID)
		}
		return thread, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		thread = &models.Thread{
			UserID:   userID,
			ThreadID: threadID,
			Title:    title,
		}
		if err := dao.Thread.Create(thread); err != nil {
			return nil, err
		}
		threadListCache.invalidate(userID, thread.ThreadID)
		return thread, nil
	}
	return nil, err
}

func (s *ThreadService) UpdateThreadTitle(userID, threadID, title string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("user_id is required")
	}
	if threadID == "" {
		return errors.New("thread_id is required")
	}
	if title == "" {
		return errors.New("title is required")
	}
	if _, err := dao.Thread.GetByUserIDAndThreadID(userID, threadID); err != nil {
		return err
	}
	if _, err := dao.Thread.UpdateTitleByUserID(userID, threadID, title); err != nil {
		return err
	}
	threadListCache.invalidate(userID, threadID)
	return nil
}

func (s *ThreadService) DeleteThread(userID, threadID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("user_id is required")
	}
	if threadID == "" {
		return errors.New("thread_id is required")
	}
	if err := dao.Thread.DeleteWithMessagesByUserID(userID, threadID); err != nil {
		return err
	}
	threadListCache.invalidate(userID, threadID)
	return nil
}

func (s *ThreadService) TouchThread(userID, threadID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("user_id is required")
	}
	if threadID == "" {
		return errors.New("thread_id is required")
	}
	if _, err := dao.Thread.GetByUserIDAndThreadID(userID, threadID); err != nil {
		return err
	}
	if _, err := dao.Thread.TouchByUserID(userID, threadID, time.Now()); err != nil {
		return err
	}
	threadListCache.invalidate(userID, threadID)
	return nil
}
