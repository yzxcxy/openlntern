package models

import (
	"time"
)

// SandboxInstance 记录每个用户当前绑定的 sandbox 实例元数据。
type SandboxInstance struct {
	ID uint `gorm:"primarykey" json:"-"`

	UserID         string    `gorm:"column:user_id;uniqueIndex:ux_sandbox_instance_user;not null;size:36" json:"user_id"`
	Provider       string    `gorm:"column:provider;index:idx_sandbox_instance_status_lease,priority:1;not null;size:32" json:"provider"`
	Status         string    `gorm:"column:status;index:idx_sandbox_instance_status_lease,priority:2;not null;size:32" json:"status"`
	InstanceID     string    `gorm:"column:instance_id;not null;size:191" json:"instance_id"`
	Endpoint       string    `gorm:"column:endpoint;not null;size:255" json:"endpoint"`
	LastActiveAt   time.Time `gorm:"column:last_active_at;not null" json:"last_active_at"`
	LeaseExpiresAt time.Time `gorm:"column:lease_expires_at;index:idx_sandbox_instance_status_lease,priority:3;not null" json:"lease_expires_at"`
	LastError      string    `gorm:"column:last_error;type:text" json:"last_error"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
