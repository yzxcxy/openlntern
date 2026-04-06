package models

import (
	"time"
)

// KBTreeEntry 知识库目录树条目，存储原始目录结构用于显示。
type KBTreeEntry struct {
	ID        uint      `gorm:"primarykey" json:"-"`
	KBID      uint      `gorm:"column:kb_id;index:ux_kb_tree_kb_path,priority:1;not null" json:"kb_id"`            // 关联KnowledgeBase.ID
	Path      string    `gorm:"column:path;index:ux_kb_tree_kb_path,priority:2;size:512;not null" json:"path"`     // 原始相对路径（相对于KB根目录）
	Name      string    `gorm:"column:name;size:255;not null" json:"name"`                                          // 文件/目录名
	IsDir     bool      `gorm:"column:is_dir;not null;default:false" json:"is_dir"`                                 // 是否为目录
	Size      int64     `gorm:"column:size;default:0" json:"size"`                                                  // 文件大小（字节）
	CreatedAt time.Time `json:"created_at"`
}