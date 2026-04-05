package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Agent struct {
	ID uint `gorm:"primarykey" json:"-"`

	UserID  string `gorm:"column:user_id;uniqueIndex:ux_agent_user_agent,priority:1;index;not null;size:36" json:"user_id"`
	AgentID string `gorm:"column:agent_id;uniqueIndex:ux_agent_user_agent,priority:2;not null;size:36" json:"agent_id"`

	Name                string `gorm:"index;not null;size:120" json:"name"`
	Description         string `gorm:"type:text" json:"description"`
	AgentType           string `gorm:"column:agent_type;index;not null;size:20" json:"agent_type"`
	Status              string `gorm:"index;not null;size:20" json:"status"`
	SystemPrompt        string `gorm:"column:system_prompt;type:longtext" json:"system_prompt"`
	AvatarURL           string `gorm:"column:avatar_url;size:255" json:"avatar_url"`
	ChatBackgroundJSON  string `gorm:"column:chat_background_json;type:longtext" json:"chat_background_json"`
	ExampleQuestionsJSON string `gorm:"column:example_questions_json;type:longtext" json:"example_questions_json"`
	DefaultModelID      string `gorm:"column:default_model_id;size:36" json:"default_model_id"`
	AgentMemoryEnabled  bool   `gorm:"column:agent_memory_enabled;not null;default:false" json:"agent_memory_enabled"`
	Sort                int    `gorm:"not null;default:0" json:"sort"`

	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

func (a *Agent) BeforeCreate(tx *gorm.DB) (err error) {
	if a.AgentID == "" {
		a.AgentID = uuid.New().String()
	}
	return nil
}
