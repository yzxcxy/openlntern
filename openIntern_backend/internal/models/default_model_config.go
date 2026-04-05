package models

import (
	"time"

	"gorm.io/gorm"
)

// DefaultModelConfig stores the globally selected default chat model.
type DefaultModelConfig struct {
	ID        uint   `gorm:"primarykey" json:"-"`
	UserID    string `gorm:"column:user_id;uniqueIndex:ux_default_model_user_key,priority:1;index;not null;size:36" json:"user_id"`
	ConfigKey string `gorm:"column:config_key;uniqueIndex:ux_default_model_user_key,priority:2;not null;size:80" json:"config_key"`
	ModelID   string `gorm:"column:model_id;size:36" json:"model_id"`

	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
