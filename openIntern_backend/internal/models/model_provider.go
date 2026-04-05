package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ModelProvider stores provider-level connection settings.
type ModelProvider struct {
	ID         uint   `gorm:"primarykey" json:"-"`
	UserID     string `gorm:"column:user_id;uniqueIndex:ux_model_provider_user_provider,priority:1;uniqueIndex:ux_model_provider_user_name,priority:1;index;not null;size:36" json:"user_id"`
	ProviderID string `gorm:"column:provider_id;uniqueIndex:ux_model_provider_user_provider,priority:2;not null;size:36" json:"provider_id"`

	Name             string `gorm:"uniqueIndex:ux_model_provider_user_name,priority:2;index;not null;size:100" json:"name"`
	APIType          string `gorm:"index;not null;size:40" json:"api_type"`
	BaseURL          string `gorm:"size:255" json:"base_url"`
	APIKeyCiphertext string `gorm:"type:text;not null" json:"-"`
	Avatar           string `gorm:"size:255" json:"avatar"`
	ExtraConfigJSON  string `gorm:"type:text" json:"extra_config_json"`
	Enabled          bool   `gorm:"not null;default:true" json:"enabled"`

	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (m *ModelProvider) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ProviderID == "" {
		m.ProviderID = uuid.New().String()
	}
	return nil
}
