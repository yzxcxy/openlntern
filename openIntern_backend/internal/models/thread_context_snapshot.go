package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ThreadContextSnapshot stores rolling compression snapshots for a chat thread.
type ThreadContextSnapshot struct {
	ID         uint   `gorm:"primarykey" json:"-"`
	SnapshotID string `gorm:"column:snapshot_id;uniqueIndex;not null;size:64" json:"snapshot_id"`
	ThreadID   string `gorm:"column:thread_id;index;not null;size:64" json:"thread_id"`

	CompressionIndex  int    `gorm:"column:compression_index;index;not null" json:"compression_index"`
	CoveredUntilMsgID string `gorm:"column:covered_until_msg_id;size:64" json:"covered_until_msg_id"`
	SummaryText       string `gorm:"type:text;not null" json:"summary_text"`
	SummaryStructJSON string `gorm:"type:text" json:"summary_struct_json"`
	ApproxTokens      int    `gorm:"column:approx_tokens;not null;default:0" json:"approx_tokens"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate assigns a stable snapshot id when the caller does not provide one.
func (s *ThreadContextSnapshot) BeforeCreate(tx *gorm.DB) (err error) {
	if s.SnapshotID == "" {
		s.SnapshotID = uuid.New().String()
	}
	return nil
}
