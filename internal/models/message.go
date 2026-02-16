package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Message struct {
	ID       uint   `gorm:"primarykey" json:"-"`
	MsgID    string `gorm:"column:msg_id;uniqueIndex;not null;size:36" json:"msg_id"`
	ThreadID string `gorm:"index;not null;size:36" json:"thread_id"`

	Type     string `gorm:"size:20" json:"type"`
	Content  string `gorm:"type:text;not null" json:"content"`
	Status   string `gorm:"size:20" json:"status"`
	Metadata string `gorm:"type:text" json:"metadata"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (m *Message) BeforeCreate(tx *gorm.DB) (err error) {
	if m.MsgID == "" {
		m.MsgID = uuid.New().String()
	}
	return
}
