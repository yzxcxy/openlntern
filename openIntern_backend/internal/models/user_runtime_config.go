package models

import (
	"encoding/json"
	"time"
)

// UserRuntimeConfig stores per-user overrides for runtime config blocks.
type UserRuntimeConfig struct {
	ID          uint            `gorm:"primarykey" json:"-"`
	UserID      string          `gorm:"column:user_id;uniqueIndex:ux_user_runtime_config_user_key,priority:1;index;not null;size:36" json:"user_id"`
	ConfigKey   string          `gorm:"column:config_key;uniqueIndex:ux_user_runtime_config_user_key,priority:2;not null;size:80" json:"config_key"`
	ConfigValue json.RawMessage `gorm:"column:config_value;type:json;not null" json:"config_value"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
