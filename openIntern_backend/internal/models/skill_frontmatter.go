package models

import (
	"time"

	"gorm.io/gorm"
)

type SkillFrontmatter struct {
	ID        uint           `gorm:"primarykey" json:"-"`
	UserID    string         `gorm:"column:user_id;index:ux_skill_frontmatter_user_name,priority:1;size:36;not null" json:"user_id"`
	SkillName string         `gorm:"column:skill_name;index:ux_skill_frontmatter_user_name,priority:2;size:255;not null" json:"skill_name"`
	Raw       string         `gorm:"type:text;not null" json:"raw"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
