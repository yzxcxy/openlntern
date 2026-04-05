package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Tool struct {
	ID uint `gorm:"primarykey" json:"-"`

	UserID           string `gorm:"column:user_id;uniqueIndex:ux_tool_user_tool,priority:1;uniqueIndex:ux_tool_user_plugin_name,priority:1;index;not null;size:36" json:"user_id"`
	ToolID           string `gorm:"column:tool_id;uniqueIndex:ux_tool_user_tool,priority:2;not null;size:36" json:"tool_id"`
	PluginID         string `gorm:"column:plugin_id;uniqueIndex:ux_tool_user_plugin_name,priority:2;index;not null;size:36" json:"plugin_id"`
	ToolName         string `gorm:"column:tool_name;uniqueIndex:ux_tool_user_plugin_name,priority:3;index;not null;size:120" json:"tool_name"`
	Description      string `gorm:"type:text" json:"description"`
	InputSchemaJSON  string `gorm:"column:input_schema_json;type:longtext" json:"input_schema_json"`
	OutputSchemaJSON string `gorm:"column:output_schema_json;type:longtext" json:"output_schema_json"`
	ToolResponseMode string `gorm:"column:tool_response_mode;size:20" json:"tool_response_mode"`
	Enabled          bool   `gorm:"not null;default:true" json:"enabled"`

	Code         string `gorm:"type:longtext" json:"code"`
	CodeLanguage string `gorm:"column:code_language;size:40" json:"code_language"`

	APIRequestType   string `gorm:"column:api_request_type;size:20" json:"api_request_type"`
	RequestURL       string `gorm:"column:request_url;size:500" json:"request_url"`
	QuerySchemaJSON  string `gorm:"column:query_schema_json;type:longtext" json:"query_schema_json"`
	HeaderSchemaJSON string `gorm:"column:header_schema_json;type:longtext" json:"header_schema_json"`
	BodySchemaJSON   string `gorm:"column:body_schema_json;type:longtext" json:"body_schema_json"`
	QueryFieldsJSON  string `gorm:"column:query_fields_json;type:longtext" json:"query_fields_json"`
	HeaderFieldsJSON string `gorm:"column:header_fields_json;type:longtext" json:"header_fields_json"`
	BodyFieldsJSON   string `gorm:"column:body_fields_json;type:longtext" json:"body_fields_json"`
	AuthConfigRef    string `gorm:"column:auth_config_ref;size:255" json:"auth_config_ref"`
	TimeoutMS        int    `gorm:"column:timeout_ms;not null;default:30000" json:"timeout_ms"`

	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

func (t *Tool) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ToolID == "" {
		t.ToolID = uuid.New().String()
	}
	return nil
}
