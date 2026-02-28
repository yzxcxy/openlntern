package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ModelCatalog stores model-level metadata independent from provider credentials.
type ModelCatalog struct {
	ID      uint   `gorm:"primarykey" json:"-"`
	ModelID string `gorm:"column:model_id;uniqueIndex;not null;size:36" json:"model_id"`

	ProviderID       string `gorm:"index;not null;size:36" json:"provider_id"`
	ModelKey         string `gorm:"index;not null;size:120" json:"model_key"`
	Name             string `gorm:"index;not null;size:120" json:"name"`
	Avatar           string `gorm:"size:255" json:"avatar"`
	CapabilitiesJSON string `gorm:"type:text" json:"capabilities_json"`
	Enabled          bool   `gorm:"not null;default:true" json:"enabled"`
	Sort             int    `gorm:"not null;default:0" json:"sort"`

	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (m *ModelCatalog) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ModelID == "" {
		m.ModelID = uuid.New().String()
	}
	return nil
}
