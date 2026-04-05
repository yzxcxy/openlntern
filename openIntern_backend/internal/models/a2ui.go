package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// A2UI 存储 A2UI 的 UI 定义和数据
type A2UI struct {
	ID     uint   `gorm:"primarykey" json:"-"`                                        // 内部自增ID，不对外暴露
	UserID string `gorm:"column:user_id;uniqueIndex:ux_a2ui_user_name,priority:1;uniqueIndex:ux_a2ui_user_a2ui,priority:1;index;not null;size:36" json:"user_id"`
	A2UIID string `gorm:"column:a2ui_id;uniqueIndex:ux_a2ui_user_a2ui,priority:2;not null;size:36" json:"a2ui_id"` // 业务ID，对外暴露为 id

	Name        string `gorm:"uniqueIndex:ux_a2ui_user_name,priority:2;not null;size:100" json:"name"` // UI 名称
	Description string `gorm:"size:255" json:"description"`         // 描述

	UIJSON   string `gorm:"type:longtext;not null" json:"ui_json"` // UI 组件结构的 JSON 字符串
	DataJSON string `gorm:"type:longtext" json:"data_json"`        // 初始数据的 JSON 字符串

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
