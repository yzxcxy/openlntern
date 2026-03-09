package models

import (
	"time"

	"gorm.io/gorm"
)

// MemoryUsageLog stores the long-term memory URIs that were actually injected for one completed run.
type MemoryUsageLog struct {
	ID         uint       `gorm:"primarykey" json:"-"`
	ThreadID   string     `gorm:"column:thread_id;uniqueIndex:ux_memory_usage_run_uri,priority:1;index;not null;size:64" json:"thread_id"`
	RunID      string     `gorm:"column:run_id;uniqueIndex:ux_memory_usage_run_uri,priority:2;index;not null;size:64" json:"run_id"`
	MemoryURI  string     `gorm:"column:memory_uri;uniqueIndex:ux_memory_usage_run_uri,priority:3;not null;size:512" json:"memory_uri"`
	ReportedAt *time.Time `gorm:"column:reported_at;index" json:"reported_at,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
