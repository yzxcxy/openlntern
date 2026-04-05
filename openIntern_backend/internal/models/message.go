package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Message struct {
	ID       uint   `gorm:"primarykey" json:"-"`
	MsgID    string `gorm:"column:msg_id;uniqueIndex;not null;size:64" json:"msg_id"`
	UserID   string `gorm:"column:user_id;index;index:idx_message_user_thread_sequence,priority:1;not null;size:36" json:"user_id"`
	ThreadID string `gorm:"index;index:idx_message_user_thread_sequence,priority:2;not null;size:64" json:"thread_id"`
	RunID    string `gorm:"index;not null;size:64" json:"run_id"`
	// Sequence 保存线程内稳定递增的消息顺序，用于历史回放，避免同一时间戳下顺序漂移。
	Sequence int64 `gorm:"index:idx_message_user_thread_sequence,priority:3;not null;default:0" json:"sequence"`

	Type     string `gorm:"size:20" json:"type"`                   // 和https://docs.ag-ui.com/concepts/messages 所拥有的类型是一样的
	Content  string `gorm:"type:longtext;not null" json:"content"` // skill/tool 结果可能远超 TEXT 上限，因此统一提升为 LONGTEXT
	Status   string `gorm:"size:20" json:"status"`
	Metadata string `gorm:"type:longtext" json:"metadata"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (m *Message) BeforeCreate(tx *gorm.DB) (err error) {
	if m.MsgID == "" {
		m.MsgID = uuid.New().String()
	}
	return
}
