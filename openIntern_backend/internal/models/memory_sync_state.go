package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	// MemorySyncStatusPending indicates the thread has new data waiting for memory sync.
	MemorySyncStatusPending = "pending"
	// MemorySyncStatusReady indicates the thread state is fully synchronized.
	MemorySyncStatusReady = "ready"
	// MemorySyncStatusFailed indicates the latest sync attempt failed and needs retry.
	MemorySyncStatusFailed = "failed"
)

const (
	// MemoryCommitStatusPending indicates add-message has advanced but commit request is not accepted yet.
	MemoryCommitStatusPending = "pending"
	// MemoryCommitStatusProcessing indicates commit request is accepted and task polling is in progress.
	MemoryCommitStatusProcessing = "processing"
	// MemoryCommitStatusCommitted indicates commit request has been accepted by OpenViking (wait=false).
	// It means async task reached completed status.
	MemoryCommitStatusCommitted = "committed"
	// MemoryCommitStatusFailed indicates commit submission failed, or async task finished with failed/invalid status.
	MemoryCommitStatusFailed = "failed"
)

// MemorySyncState stores the long-term memory sync cursor for a chat thread.
type MemorySyncState struct {
	ID                 uint       `gorm:"primarykey" json:"-"`
	ThreadID           string     `gorm:"column:thread_id;uniqueIndex;not null;size:64" json:"thread_id"`
	LastAddedMsgID     string     `gorm:"column:last_added_msg_id;size:64" json:"last_added_msg_id"`
	LastSyncedMsgID    string     `gorm:"column:last_synced_msg_id;size:64" json:"last_synced_msg_id"`
	LastCommittedRunID string     `gorm:"column:last_committed_run_id;size:64" json:"last_committed_run_id"`
	CommitTaskID       string     `gorm:"column:commit_task_id;size:64" json:"commit_task_id"`
	CommitTaskStatus   string     `gorm:"column:commit_task_status;size:32" json:"commit_task_status"`
	CommitStatus       string     `gorm:"column:commit_status;size:32;not null;default:'pending'" json:"commit_status"`
	Status             string     `gorm:"column:status;size:32;not null;default:'pending'" json:"status"`
	RetryCount         int        `gorm:"column:retry_count;not null;default:0" json:"retry_count"`
	LastError          string     `gorm:"column:last_error;type:text" json:"last_error"`
	NextAttemptAt      *time.Time `gorm:"column:next_attempt_at;index" json:"next_attempt_at"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
