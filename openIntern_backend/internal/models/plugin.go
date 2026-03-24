package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Plugin struct {
	ID uint `gorm:"primarykey" json:"-"`

	PluginID    string     `gorm:"column:plugin_id;uniqueIndex;not null;size:36" json:"plugin_id"`
	Name        string     `gorm:"index;not null;size:120" json:"name"`
	Description string     `gorm:"type:text" json:"description"`
	Icon        string     `gorm:"size:255" json:"icon"`
	Source      string     `gorm:"index;not null;size:20" json:"source"`
	RuntimeType string     `gorm:"index;not null;size:20" json:"runtime_type"`
	Status      string     `gorm:"index;not null;size:20" json:"status"`
	MCPURL      string     `gorm:"column:mcp_url;size:255" json:"mcp_url"`
	MCPProtocol string     `gorm:"column:mcp_protocol;size:40" json:"mcp_protocol"`
	TimeoutMS   int        `gorm:"column:timeout_ms;default:30000" json:"timeout_ms"`
	LastSyncAt  *time.Time `gorm:"column:last_sync_at" json:"last_sync_at"`

	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

func (p *Plugin) BeforeCreate(tx *gorm.DB) (err error) {
	if p.PluginID == "" {
		p.PluginID = uuid.New().String()
	}
	return nil
}
