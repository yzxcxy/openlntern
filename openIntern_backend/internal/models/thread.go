package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Thread struct {
	ID       uint   `gorm:"primarykey" json:"-"`
	ThreadID string `gorm:"column:thread_id;uniqueIndex;not null;size:36" json:"thread_id"`
	Title    string `gorm:"size:200" json:"title"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (t *Thread) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ThreadID == "" {
		t.ThreadID = uuid.New().String()
	}
	return
}
