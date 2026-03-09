package models

import (
	"time"

	"gorm.io/gorm"
)

type PluginDefault struct {
	ID uint `gorm:"primarykey" json:"-"`

	ModelID string `gorm:"column:model_id;index;not null;size:64" json:"model_id"`
	ToolID  string `gorm:"column:tool_id;index;not null;size:36" json:"tool_id"`

	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}
