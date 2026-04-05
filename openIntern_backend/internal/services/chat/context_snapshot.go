package chat

import (
	"errors"
	"strings"

	"openIntern/internal/dao"
	"openIntern/internal/models"
)

// ThreadContextSnapshotService manages thread context compression snapshots.
type ThreadContextSnapshotService struct{}

// ThreadContextSnapshot is the shared service singleton for snapshot access.
var ThreadContextSnapshot = new(ThreadContextSnapshotService)

// GetLatestByThreadID reads the latest snapshot by thread id.
func (s *ThreadContextSnapshotService) GetLatestByThreadID(userID, threadID string) (*models.ThreadContextSnapshot, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user_id is required")
	}
	if strings.TrimSpace(threadID) == "" {
		return nil, errors.New("thread_id is required")
	}
	return dao.ThreadContextSnapshot.GetLatestByUserIDAndThreadID(strings.TrimSpace(userID), strings.TrimSpace(threadID))
}

// Create validates and persists a thread context snapshot.
func (s *ThreadContextSnapshotService) Create(item *models.ThreadContextSnapshot) error {
	if item == nil {
		return errors.New("snapshot is required")
	}
	item.ThreadID = strings.TrimSpace(item.ThreadID)
	item.UserID = strings.TrimSpace(item.UserID)
	item.CoveredUntilMsgID = strings.TrimSpace(item.CoveredUntilMsgID)
	item.SummaryText = strings.TrimSpace(item.SummaryText)
	item.SummaryStructJSON = strings.TrimSpace(item.SummaryStructJSON)
	if item.UserID == "" {
		return errors.New("user_id is required")
	}
	if item.ThreadID == "" {
		return errors.New("thread_id is required")
	}
	if item.SummaryText == "" {
		return errors.New("summary_text is required")
	}
	if item.CompressionIndex <= 0 {
		return errors.New("compression_index must be greater than 0")
	}
	if item.ApproxTokens < 0 {
		item.ApproxTokens = 0
	}
	return dao.ThreadContextSnapshot.Create(item)
}
