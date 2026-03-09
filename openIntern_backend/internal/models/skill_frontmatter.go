package models

import (
	"time"

	"gorm.io/gorm"
)

type SkillFrontmatter struct {
	ID        uint           `gorm:"primarykey" json:"-"`
	SkillName string         `gorm:"index;size:255;not null" json:"skill_name"`
	Raw       string         `gorm:"type:text;not null" json:"raw"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
