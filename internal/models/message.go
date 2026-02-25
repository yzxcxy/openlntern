package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Message struct {
	ID       uint   `gorm:"primarykey" json:"-"`
	MsgID    string `gorm:"column:msg_id;uniqueIndex;not null;size:64" json:"msg_id"`
	ThreadID string `gorm:"index;not null;size:64" json:"thread_id"`
	RunID    string `gorm:"index;not null;size:64" json:"run_id"`

	Type     string `gorm:"size:20" json:"type"`  // 和https://docs.ag-ui.com/concepts/messages 所拥有的类型是一样的
	Content  string `gorm:"type:text;not null" json:"content"` // 这里面存放的是AGUI的消息结构体：https://docs.ag-ui.com/concepts/messages
	Status   string `gorm:"size:20" json:"status"` 
	Metadata string `gorm:"type:text" json:"metadata"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (m *Message) BeforeCreate(tx *gorm.DB) (err error) {
	if m.MsgID == "" {
		m.MsgID = uuid.New().String()
	}
	return
}
