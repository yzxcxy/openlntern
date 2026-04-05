package models

import (
	"time"

	"gorm.io/gorm"
)

type PluginDefault struct {
	ID uint `gorm:"primarykey" json:"-"`

	UserID  string `gorm:"column:user_id;uniqueIndex:ux_plugin_default_user_model_tool,priority:1;index;not null;size:36" json:"user_id"`
	ModelID string `gorm:"column:model_id;uniqueIndex:ux_plugin_default_user_model_tool,priority:2;index;not null;size:64" json:"model_id"`
	ToolID  string `gorm:"column:tool_id;uniqueIndex:ux_plugin_default_user_model_tool,priority:3;index;not null;size:36" json:"tool_id"`

	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}
