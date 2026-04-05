package models

import (
	"time"

	"gorm.io/gorm"
)

type AgentBinding struct {
	ID uint `gorm:"primarykey" json:"-"`

	UserID          string `gorm:"column:user_id;uniqueIndex:idx_agent_binding_unique,priority:1;index;not null;size:36" json:"user_id"`
	AgentID         string `gorm:"column:agent_id;uniqueIndex:idx_agent_binding_unique,priority:2;index;not null;size:36" json:"agent_id"`
	BindingType     string `gorm:"column:binding_type;uniqueIndex:idx_agent_binding_unique,priority:3;index;not null;size:20" json:"binding_type"`
	BindingTargetID string `gorm:"column:binding_target_id;uniqueIndex:idx_agent_binding_unique,priority:4;index;not null;size:255" json:"binding_target_id"`
	Sort            int    `gorm:"not null;default:0" json:"sort"`
	MetadataJSON    string `gorm:"column:metadata_json;type:longtext" json:"metadata_json"`

	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}
