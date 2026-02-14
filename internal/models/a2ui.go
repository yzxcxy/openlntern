package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// A2UIType 定义 A2UI 的类型
type A2UIType string

const (
	A2UITypeOfficial A2UIType = "official" // 官方预定义
	A2UITypeCustom   A2UIType = "custom"   // 用户自定义
)

// A2UI 存储 A2UI 的 UI 定义和数据
type A2UI struct {
	ID     uint   `gorm:"primarykey" json:"-"`                         // 内部自增ID，不对外暴露
	A2UIID string `gorm:"column:a2ui_id;uniqueIndex;not null;size:36" json:"a2ui_id"` // 业务ID，对外暴露为 id

	Name        string `gorm:"index;not null;size:100" json:"name"` // UI 名称
	Description string `gorm:"size:255" json:"description"`         // 描述

	Type A2UIType `gorm:"size:20;default:'custom';not null" json:"type"` // 类型：official 或 custom

	UIJSON   string `gorm:"type:text;not null" json:"ui_json"` // UI 组件结构的 JSON 字符串
	DataJSON string `gorm:"type:text" json:"data_json"`        // 初始数据的 JSON 字符串

	UserID uint `gorm:"index" json:"user_id"` // 所属用户 ID

	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate GORM hook to set A2UIID if not present
func (a *A2UI) BeforeCreate(tx *gorm.DB) (err error) {
	if a.A2UIID == "" {
		a.A2UIID = uuid.New().String()
	}
	return
}
