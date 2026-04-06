package models

import (
	"time"

	"gorm.io/gorm"
)

// KnowledgeBase 知识库元数据，存储KB基本信息和双存储路径。
type KnowledgeBase struct {
	ID            uint           `gorm:"primarykey" json:"-"`
	UserID        string         `gorm:"column:user_id;uniqueIndex:ux_kb_user_name,priority:1;size:36;not null" json:"user_id"`
	Name          string         `gorm:"column:name;uniqueIndex:ux_kb_user_name,priority:2;size:255;not null" json:"name"`
	OpenVikingURI string         `gorm:"column:openviking_uri;size:512" json:"openviking_uri"` // OpenViking中的URI，用于Agent检索
	LocalPath     string         `gorm:"column:local_path;size:512" json:"local_path"`           // 本地MinIO存储路径前缀，用于预览
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}