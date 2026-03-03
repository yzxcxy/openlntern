package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"openIntern/internal/config"
	"openIntern/internal/dao"
	"openIntern/internal/models"
	toolsvc "openIntern/internal/services/tools"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	pluginSourceCustom  = "custom"
	pluginSourceBuiltin = "builtin"

	pluginRuntimeAPI  = "api"
	pluginRuntimeMCP  = "mcp"
	pluginRuntimeCode = "code"

	mcpProtocolSSE            = "sse"
	mcpProtocolStreamableHTTP = "streamableHttp"

	pluginStatusEnabled  = "enabled"
	pluginStatusDisabled = "disabled"

	apiMethodGET  = "GET"
	apiMethodPOST = "POST"

	codeLanguagePython     = "python"
	codeLanguageJavaScript = "javascript"

	toolResponseStreaming    = "streaming"
	toolResponseNonStreaming = "non_streaming"

	defaultPluginTimeoutMS = 30000
)

type PluginFieldInput struct {
	Name         string             `json:"name"`
	Type         string             `json:"type"`
	Required     bool               `json:"required"`
	Description  string             `json:"description"`
	DefaultValue any                `json:"default_value,omitempty"`
	EnumValues   []string           `json:"enum_values"`
	Children     []PluginFieldInput `json:"children"`
	Items        *PluginFieldInput  `json:"items"`
}

type PluginToolInput struct {
	ToolID           string             `json:"tool_id"`
	ToolName         string             `json:"tool_name"`
	Description      string             `json:"description"`
	ToolResponseMode string             `json:"tool_response_mode"`
	OutputSchemaJSON string             `json:"output_schema_json"`
	Enabled          *bool              `json:"enabled"`
	APIRequestType   string             `json:"api_request_type"`
	RequestURL       string             `json:"request_url"`
	QueryFields      []PluginFieldInput `json:"query_fields"`
	HeaderFields     []PluginFieldInput `json:"header_fields"`
	BodyFields       []PluginFieldInput `json:"body_fields"`
	AuthConfigRef    string             `json:"auth_config_ref"`
	TimeoutMS        int                `json:"timeout_ms"`
	Code             string             `json:"code"`
	CodeLanguage     string             `json:"code_language"`
}

type UpsertPluginInput struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Icon        string            `json:"icon"`
	Source      string            `json:"source"`
	RuntimeType string            `json:"runtime_type"`
	Enabled     *bool             `json:"enabled"`
	MCPURL      string            `json:"mcp_url"`
	MCPProtocol string            `json:"mcp_protocol"`
	Tools       []PluginToolInput `json:"tools"`
}

type PluginToolView struct {
	ToolID           string             `json:"tool_id"`
	ToolName         string             `json:"tool_name"`
	Description      string             `json:"description"`
	InputSchemaJSON  string             `json:"input_schema_json"`
	OutputSchemaJSON string             `json:"output_schema_json"`
	ToolResponseMode string             `json:"tool_response_mode"`
	Enabled          bool               `json:"enabled"`
	APIRequestType   string             `json:"api_request_type"`
	RequestURL       string             `json:"request_url"`
	QueryFields      []PluginFieldInput `json:"query_fields"`
	HeaderFields     []PluginFieldInput `json:"header_fields"`
	BodyFields       []PluginFieldInput `json:"body_fields"`
	AuthConfigRef    string             `json:"auth_config_ref"`
	TimeoutMS        int                `json:"timeout_ms"`
	CodeLanguage     string             `json:"code_language"`
	Code             string             `json:"code"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

type PluginView struct {
	PluginID    string           `json:"plugin_id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Icon        string           `json:"icon"`
	Source      string           `json:"source"`
	RuntimeType string           `json:"runtime_type"`
	Status      string           `json:"status"`
	MCPURL      string           `json:"mcp_url"`
	MCPProtocol string           `json:"mcp_protocol"`
	LastSyncAt  *time.Time       `json:"last_sync_at"`
	ToolCount   int              `json:"tool_count"`
	Tools       []PluginToolView `json:"tools"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

type PluginListFilter struct {
	Source      string
	RuntimeType string
	Status      string
	Keyword     string
}

type ChatToolView struct {
	ToolID           string `json:"tool_id"`
	ToolName         string `json:"tool_name"`
	Description      string `json:"description"`
	ToolResponseMode string `json:"tool_response_mode"`
}

type ChatPluginView struct {
	PluginID    string         `json:"plugin_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Icon        string         `json:"icon"`
	Source      string         `json:"source"`
	RuntimeType string         `json:"runtime_type"`
	Tools       []ChatToolView `json:"tools"`
}

type PluginService struct{}

var Plugin = new(PluginService)
var pluginDefaultIconURL string

func InitPlugin(cfg config.PluginConfig) {
	pluginDefaultIconURL = strings.TrimSpace(cfg.DefaultIconURL)
	initPluginMCPSync(cfg)
}

func GetDefaultPluginIconURL() string {
	return pluginDefaultIconURL
}

func (s *PluginService) Create(input UpsertPluginInput) (*PluginView, error) {
	plugin, tools, err := s.prepareDefinition("", input, nil)
	if err != nil {
		return nil, err
	}
	if err := dao.Plugin.Create(plugin, tools); err != nil {
		return nil, err
	}
	if plugin.RuntimeType == pluginRuntimeMCP {
		if err := s.queueMCPPluginSync(plugin.PluginID, mcpSyncDelay); err != nil {
			log.Printf("enqueue mcp sync after create failed plugin_id=%s err=%v", plugin.PluginID, err)
		}
	}
	return s.GetByPluginID(plugin.PluginID)
}

func (s *PluginService) Update(pluginID string, input UpsertPluginInput) (*PluginView, error) {
	existing, err := s.getPluginRecord(pluginID)
	if err != nil {
		return nil, err
	}
	plugin, tools, err := s.prepareDefinition(pluginID, input, existing)
	if err != nil {
		return nil, err
	}
	if err := dao.Plugin.Update(pluginID, plugin, tools); err != nil {
		return nil, err
	}
	if plugin.RuntimeType == pluginRuntimeMCP {
		if err := s.queueMCPPluginSync(plugin.PluginID, mcpSyncDelay); err != nil {
			log.Printf("enqueue mcp sync after update failed plugin_id=%s err=%v", plugin.PluginID, err)
		}
	} else {
		s.clearMCPPluginSync(pluginID)
	}
	return s.GetByPluginID(pluginID)
}

func (s *PluginService) GetByPluginID(pluginID string) (*PluginView, error) {
	plugin, err := s.getPluginRecord(pluginID)
	if err != nil {
		return nil, err
	}
	toolMap, err := s.loadToolMap([]string{pluginID})
	if err != nil {
		return nil, err
	}
	view := buildPluginView(*plugin, toolMap[pluginID])
	return &view, nil
}

func (s *PluginService) List(page, pageSize int, filter PluginListFilter) ([]PluginView, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	items, total, err := dao.Plugin.List(page, pageSize, dao.PluginListFilter{
		Source:      normalizeOptionalFilter(filter.Source),
		RuntimeType: normalizeOptionalFilter(filter.RuntimeType),
		Status:      normalizeOptionalFilter(filter.Status),
		Keyword:     strings.TrimSpace(filter.Keyword),
	})
	if err != nil {
		return nil, 0, err
	}
	pluginIDs := make([]string, 0, len(items))
	for _, item := range items {
		pluginIDs = append(pluginIDs, item.PluginID)
	}
	toolMap, err := s.loadToolMap(pluginIDs)
	if err != nil {
		return nil, 0, err
	}
	views := make([]PluginView, 0, len(items))
	for _, item := range items {
		views = append(views, buildPluginView(item, toolMap[item.PluginID]))
	}
	return views, total, nil
}

func (s *PluginService) Delete(pluginID string) error {
	plugin, err := s.getPluginRecord(pluginID)
	if err != nil {
		return err
	}
	if plugin.Source == pluginSourceBuiltin {
		return errors.New("builtin plugin is read-only")
	}
	if err := dao.Plugin.Delete(pluginID); err != nil {
		return err
	}
	s.clearMCPPluginSync(pluginID)
	return nil
}

func (s *PluginService) SetEnabled(pluginID string, enabled bool) (*PluginView, error) {
	plugin, err := s.getPluginRecord(pluginID)
	if err != nil {
		return nil, err
	}
	if plugin.Source == pluginSourceBuiltin {
		return nil, errors.New("builtin plugin is read-only")
	}
	status := pluginStatusDisabled
	if enabled {
		status = pluginStatusEnabled
	}
	if err := dao.Plugin.UpdateStatus(pluginID, status); err != nil {
		return nil, err
	}
	if plugin.RuntimeType == pluginRuntimeMCP {
		if enabled {
			if err := s.queueMCPPluginSync(pluginID, 0); err != nil {
				log.Printf("enqueue mcp sync after enable failed plugin_id=%s err=%v", pluginID, err)
			}
		} else {
			s.clearMCPPluginSync(pluginID)
		}
	}
	return s.GetByPluginID(pluginID)
}

func (s *PluginService) Sync(pluginID string) (*PluginView, error) {
	plugin, err := s.getPluginRecord(pluginID)
	if err != nil {
		return nil, err
	}
	if plugin.RuntimeType != pluginRuntimeMCP {
		return nil, errors.New("only mcp plugins support sync")
	}
	if err := s.syncMCPPluginNow(context.Background(), pluginID, true); err != nil {
		return nil, err
	}
	return s.GetByPluginID(pluginID)
}

func (s *PluginService) ListAvailableForChat() ([]ChatPluginView, error) {
	plugins, err := dao.Plugin.ListEnabled()
	if err != nil {
		return nil, err
	}
	pluginIDs := make([]string, 0, len(plugins))
	for _, item := range plugins {
		pluginIDs = append(pluginIDs, item.PluginID)
	}
	toolMap, err := s.loadToolMap(pluginIDs)
	if err != nil {
		return nil, err
	}
	results := make([]ChatPluginView, 0, len(plugins))
	for _, plugin := range plugins {
		tools := toolMap[plugin.PluginID]
		lightTools := make([]ChatToolView, 0, len(tools))
		for _, tool := range tools {
			if !tool.Enabled {
				continue
			}
			lightTools = append(lightTools, ChatToolView{
				ToolID:           tool.ToolID,
				ToolName:         tool.ToolName,
				Description:      tool.Description,
				ToolResponseMode: tool.ToolResponseMode,
			})
		}
		results = append(results, ChatPluginView{
			PluginID:    plugin.PluginID,
			Name:        plugin.Name,
			Description: plugin.Description,
			Icon:        normalizePluginIcon(plugin.Icon),
			Source:      plugin.Source,
			RuntimeType: plugin.RuntimeType,
			Tools:       lightTools,
		})
	}
	return results, nil
}

func (s *PluginService) getPluginRecord(pluginID string) (*models.Plugin, error) {
	return dao.Plugin.GetByPluginID(pluginID)
}

func (s *PluginService) prepareDefinition(pluginID string, input UpsertPluginInput, existing *models.Plugin) (*models.Plugin, []models.Tool, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, nil, errors.New("name is required")
	}

	source := normalizePluginSource(input.Source)
	if strings.TrimSpace(input.Source) != "" && source == "" {
		return nil, nil, errors.New("source must be custom")
	}
	if existing == nil {
		if source == "" {
			source = pluginSourceCustom
		}
		if source == pluginSourceBuiltin {
			return nil, nil, errors.New("builtin plugin is read-only")
		}
	} else {
		if existing.Source == pluginSourceBuiltin {
			return nil, nil, errors.New("builtin plugin is read-only")
		}
		if source == "" {
			source = existing.Source
		}
		if source == pluginSourceBuiltin {
			return nil, nil, errors.New("builtin plugin is read-only")
		}
	}
	if source != pluginSourceCustom {
		return nil, nil, errors.New("source must be custom")
	}

	runtimeType := normalizeRuntimeType(input.RuntimeType)
	if strings.TrimSpace(input.RuntimeType) != "" && runtimeType == "" {
		return nil, nil, errors.New("unsupported runtime_type")
	}
	if runtimeType == "" {
		if existing != nil {
			runtimeType = existing.RuntimeType
		}
	}
	if runtimeType == "" {
		return nil, nil, errors.New("runtime_type is required")
	}

	status := pluginStatusEnabled
	if input.Enabled != nil && !*input.Enabled {
		status = pluginStatusDisabled
	}
	if existing != nil && input.Enabled == nil {
		status = existing.Status
	}

	nextPluginID := pluginID
	if existing != nil {
		nextPluginID = existing.PluginID
	}
	if strings.TrimSpace(nextPluginID) == "" {
		nextPluginID = uuid.New().String()
	}

	plugin := &models.Plugin{
		PluginID:    nextPluginID,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		Icon:        normalizePluginIcon(input.Icon),
		Source:      source,
		RuntimeType: runtimeType,
		Status:      status,
		MCPURL:      "",
		MCPProtocol: "",
		LastSyncAt:  nil,
	}
	if existing != nil {
		plugin.ID = existing.ID
		plugin.CreatedAt = existing.CreatedAt
	}

	if runtimeType == pluginRuntimeMCP {
		mcpURL := strings.TrimSpace(input.MCPURL)
		if mcpURL == "" {
			return nil, nil, errors.New("mcp_url is required for mcp plugins")
		}
		if err := validateHTTPURL(mcpURL); err != nil {
			return nil, nil, err
		}
		mcpProtocol := normalizeMCPProtocol(input.MCPProtocol)
		if mcpProtocol == "" {
			return nil, nil, errors.New("mcp_protocol must be sse or streamableHttp")
		}
		plugin.MCPURL = mcpURL
		plugin.MCPProtocol = mcpProtocol
		if existing != nil {
			plugin.LastSyncAt = existing.LastSyncAt
		}
	}

	tools, err := s.buildTools(nextPluginID, runtimeType, input.Tools)
	if err != nil {
		return nil, nil, err
	}
	return plugin, tools, nil
}

func (s *PluginService) buildTools(pluginID string, runtimeType string, inputs []PluginToolInput) ([]models.Tool, error) {
	if runtimeType == pluginRuntimeMCP {
		if len(inputs) == 0 {
			return []models.Tool{}, nil
		}
	}
	if (runtimeType == pluginRuntimeAPI || runtimeType == pluginRuntimeCode) && len(inputs) == 0 {
		return nil, errors.New("at least one tool is required")
	}
	tools := make([]models.Tool, 0, len(inputs))
	seenToolNames := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		tool, err := s.buildTool(pluginID, runtimeType, input)
		if err != nil {
			return nil, err
		}
		if _, exists := seenToolNames[tool.ToolName]; exists {
			return nil, errors.New("tool names must be unique")
		}
		seenToolNames[tool.ToolName] = struct{}{}
		tools = append(tools, tool)
	}
	return tools, nil
}

func (s *PluginService) buildTool(pluginID string, runtimeType string, input PluginToolInput) (models.Tool, error) {
	toolName := strings.TrimSpace(input.ToolName)
	if toolName == "" {
		return models.Tool{}, errors.New("tool_name is required")
	}
	if runtimeType != pluginRuntimeMCP && !toolsvc.IsModelSafeToolName(toolName) {
		return models.Tool{}, errors.New("tool_name must match ^[a-zA-Z0-9_-]+$")
	}

	toolID := strings.TrimSpace(input.ToolID)
	if toolID == "" {
		toolID = uuid.New().String()
	}

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	outputSchemaJSON, err := normalizeOutputSchema(input.OutputSchemaJSON)
	if err != nil {
		return models.Tool{}, err
	}

	tool := models.Tool{
		ToolID:           toolID,
		PluginID:         pluginID,
		ToolName:         toolName,
		Description:      strings.TrimSpace(input.Description),
		OutputSchemaJSON: outputSchemaJSON,
		Enabled:          enabled,
		TimeoutMS:        defaultPluginTimeoutMS,
	}

	if input.TimeoutMS > 0 {
		tool.TimeoutMS = input.TimeoutMS
	}
	if tool.TimeoutMS <= 0 {
		return models.Tool{}, errors.New("timeout_ms must be greater than 0")
	}

	switch runtimeType {
	case pluginRuntimeAPI:
		responseMode := normalizeToolResponseMode(input.ToolResponseMode)
		if responseMode == "" {
			return models.Tool{}, errors.New("tool_response_mode is required for api tools")
		}
		requestType := normalizeAPIRequestType(input.APIRequestType)
		if requestType == "" {
			return models.Tool{}, errors.New("api_request_type is required for api tools")
		}
		if err := validateHTTPURL(input.RequestURL); err != nil {
			return models.Tool{}, err
		}
		if requestType == apiMethodGET && len(input.BodyFields) > 0 {
			return models.Tool{}, errors.New("GET api tools do not support body fields")
		}
		queryFields, queryFieldsJSON, querySchemaJSON, err := normalizeFields(input.QueryFields)
		if err != nil {
			return models.Tool{}, err
		}
		headerFields, headerFieldsJSON, headerSchemaJSON, err := normalizeFields(input.HeaderFields)
		if err != nil {
			return models.Tool{}, err
		}
		bodyFields, bodyFieldsJSON, bodySchemaJSON, err := normalizeFields(input.BodyFields)
		if err != nil {
			return models.Tool{}, err
		}
		if requestType == apiMethodGET {
			bodyFields = []PluginFieldInput{}
			bodyFieldsJSON = "[]"
			bodySchemaJSON = ""
		}
		inputSchemaJSON, err := buildCombinedInputSchema(queryFields, headerFields, bodyFields)
		if err != nil {
			return models.Tool{}, err
		}
		tool.ToolResponseMode = responseMode
		tool.APIRequestType = requestType
		tool.RequestURL = strings.TrimSpace(input.RequestURL)
		tool.QueryFieldsJSON = queryFieldsJSON
		tool.HeaderFieldsJSON = headerFieldsJSON
		tool.BodyFieldsJSON = bodyFieldsJSON
		tool.QuerySchemaJSON = querySchemaJSON
		tool.HeaderSchemaJSON = headerSchemaJSON
		tool.BodySchemaJSON = bodySchemaJSON
		tool.InputSchemaJSON = inputSchemaJSON
		tool.AuthConfigRef = strings.TrimSpace(input.AuthConfigRef)
	case pluginRuntimeCode:
		if mode := normalizeToolResponseMode(input.ToolResponseMode); mode != "" && mode != toolResponseNonStreaming {
			return models.Tool{}, errors.New("code tools only support non_streaming")
		}
		code := strings.TrimSpace(input.Code)
		if code == "" {
			return models.Tool{}, errors.New("code is required for code tools")
		}
		codeLanguage := normalizeCodeLanguage(input.CodeLanguage)
		if codeLanguage == "" {
			return models.Tool{}, errors.New("code_language must be python or javascript")
		}
		bodyFields, bodyFieldsJSON, bodySchemaJSON, err := normalizeFields(input.BodyFields)
		if err != nil {
			return models.Tool{}, err
		}
		inputSchemaJSON, err := marshalJSONString(buildObjectSchema(bodyFields))
		if err != nil {
			return models.Tool{}, err
		}
		tool.ToolResponseMode = toolResponseNonStreaming
		tool.Code = code
		tool.CodeLanguage = codeLanguage
		tool.BodyFieldsJSON = bodyFieldsJSON
		tool.BodySchemaJSON = bodySchemaJSON
		tool.InputSchemaJSON = inputSchemaJSON
	case pluginRuntimeMCP:
		if mode := normalizeToolResponseMode(input.ToolResponseMode); mode != "" {
			tool.ToolResponseMode = mode
		}
	default:
		return models.Tool{}, errors.New("unsupported runtime_type")
	}

	return tool, nil
}

func (s *PluginService) loadToolMap(pluginIDs []string) (map[string][]models.Tool, error) {
	return dao.Plugin.LoadToolMap(pluginIDs)
}

func normalizePluginIcon(value string) string {
	icon := strings.TrimSpace(value)
	if icon == "" {
		return pluginDefaultIconURL
	}
	return icon
}

func diffPluginTools(existing []models.Tool, incoming []models.Tool) ([]models.Tool, []models.Tool, []string) {
	existingByID := make(map[string]models.Tool, len(existing))
	for _, tool := range existing {
		existingByID[tool.ToolID] = tool
	}

	toolsToUpdate := make([]models.Tool, 0, len(incoming))
	toolsToCreate := make([]models.Tool, 0, len(incoming))
	seenIncoming := make(map[string]struct{}, len(incoming))

	for _, tool := range incoming {
		if _, ok := existingByID[tool.ToolID]; ok {
			toolsToUpdate = append(toolsToUpdate, tool)
		} else {
			toolsToCreate = append(toolsToCreate, tool)
		}
		seenIncoming[tool.ToolID] = struct{}{}
	}

	removedToolIDs := make([]string, 0, len(existing))
	for _, tool := range existing {
		if _, ok := seenIncoming[tool.ToolID]; ok {
			continue
		}
		removedToolIDs = append(removedToolIDs, tool.ToolID)
	}

	return toolsToUpdate, toolsToCreate, removedToolIDs
}

func buildToolUpdateMap(tool models.Tool) map[string]any {
	return map[string]any{
		"plugin_id":          tool.PluginID,
		"tool_name":          tool.ToolName,
		"description":        tool.Description,
		"input_schema_json":  tool.InputSchemaJSON,
		"output_schema_json": tool.OutputSchemaJSON,
		"tool_response_mode": tool.ToolResponseMode,
		"enabled":            tool.Enabled,
		"code":               tool.Code,
		"code_language":      tool.CodeLanguage,
		"api_request_type":   tool.APIRequestType,
		"request_url":        tool.RequestURL,
		"query_schema_json":  tool.QuerySchemaJSON,
		"header_schema_json": tool.HeaderSchemaJSON,
		"body_schema_json":   tool.BodySchemaJSON,
		"query_fields_json":  tool.QueryFieldsJSON,
		"header_fields_json": tool.HeaderFieldsJSON,
		"body_fields_json":   tool.BodyFieldsJSON,
		"auth_config_ref":    tool.AuthConfigRef,
		"timeout_ms":         tool.TimeoutMS,
	}
}

func buildPluginView(plugin models.Plugin, tools []models.Tool) PluginView {
	toolViews := make([]PluginToolView, 0, len(tools))
	for _, tool := range tools {
		toolViews = append(toolViews, PluginToolView{
			ToolID:           tool.ToolID,
			ToolName:         tool.ToolName,
			Description:      tool.Description,
			InputSchemaJSON:  tool.InputSchemaJSON,
			OutputSchemaJSON: tool.OutputSchemaJSON,
			ToolResponseMode: tool.ToolResponseMode,
			Enabled:          tool.Enabled,
			APIRequestType:   tool.APIRequestType,
			RequestURL:       tool.RequestURL,
			QueryFields:      decodeFieldList(tool.QueryFieldsJSON),
			HeaderFields:     decodeFieldList(tool.HeaderFieldsJSON),
			BodyFields:       decodeFieldList(tool.BodyFieldsJSON),
			AuthConfigRef:    tool.AuthConfigRef,
			TimeoutMS:        tool.TimeoutMS,
			CodeLanguage:     tool.CodeLanguage,
			Code:             tool.Code,
			CreatedAt:        tool.CreatedAt,
			UpdatedAt:        tool.UpdatedAt,
		})
	}
	return PluginView{
		PluginID:    plugin.PluginID,
		Name:        plugin.Name,
		Description: plugin.Description,
		Icon:        normalizePluginIcon(plugin.Icon),
		Source:      plugin.Source,
		RuntimeType: plugin.RuntimeType,
		Status:      plugin.Status,
		MCPURL:      plugin.MCPURL,
		MCPProtocol: plugin.MCPProtocol,
		LastSyncAt:  plugin.LastSyncAt,
		ToolCount:   len(toolViews),
		Tools:       toolViews,
		CreatedAt:   plugin.CreatedAt,
		UpdatedAt:   plugin.UpdatedAt,
	}
}

func normalizeFields(fields []PluginFieldInput) ([]PluginFieldInput, string, string, error) {
	normalized := make([]PluginFieldInput, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		next, err := normalizeField(field, true)
		if err != nil {
			return nil, "", "", err
		}
		if _, exists := seen[next.Name]; exists {
			return nil, "", "", errors.New("field names must be unique")
		}
		seen[next.Name] = struct{}{}
		normalized = append(normalized, next)
	}
	fieldsJSON, err := marshalJSONString(normalized)
	if err != nil {
		return nil, "", "", err
	}
	if len(normalized) == 0 {
		return normalized, fieldsJSON, "", nil
	}
	schemaJSON, err := marshalJSONString(buildObjectSchema(normalized))
	if err != nil {
		return nil, "", "", err
	}
	return normalized, fieldsJSON, schemaJSON, nil
}

func normalizeField(input PluginFieldInput, requireName bool) (PluginFieldInput, error) {
	name := strings.TrimSpace(input.Name)
	if requireName && name == "" {
		return PluginFieldInput{}, errors.New("field name is required")
	}
	fieldType := strings.ToLower(strings.TrimSpace(input.Type))
	switch fieldType {
	case "string", "number", "integer", "boolean", "object", "array":
	default:
		return PluginFieldInput{}, errors.New("unsupported field type")
	}
	field := PluginFieldInput{
		Name:        name,
		Type:        fieldType,
		Required:    input.Required,
		Description: strings.TrimSpace(input.Description),
	}
	if input.DefaultValue != nil {
		defaultValue, err := normalizeFieldDefaultValue(input.DefaultValue, fieldType)
		if err != nil {
			return PluginFieldInput{}, err
		}
		field.DefaultValue = defaultValue
	}
	for _, value := range input.EnumValues {
		value = strings.TrimSpace(value)
		if value != "" {
			field.EnumValues = append(field.EnumValues, value)
		}
	}
	switch fieldType {
	case "object":
		seen := make(map[string]struct{}, len(input.Children))
		children := make([]PluginFieldInput, 0, len(input.Children))
		for _, child := range input.Children {
			next, err := normalizeField(child, true)
			if err != nil {
				return PluginFieldInput{}, err
			}
			if _, exists := seen[next.Name]; exists {
				return PluginFieldInput{}, errors.New("field names must be unique")
			}
			seen[next.Name] = struct{}{}
			children = append(children, next)
		}
		field.Children = children
	case "array":
		if input.Items == nil {
			return PluginFieldInput{}, errors.New("array fields require items")
		}
		item, err := normalizeField(*input.Items, false)
		if err != nil {
			return PluginFieldInput{}, err
		}
		field.Items = &item
	default:
		if len(field.EnumValues) > 0 && fieldType != "string" {
			return PluginFieldInput{}, errors.New("enum_values only support string fields")
		}
	}
	return field, nil
}

func buildObjectSchema(fields []PluginFieldInput) map[string]any {
	properties, required := buildSchemaFields(fields)
	schema := map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func buildFieldObjectSchema(fields []PluginFieldInput) map[string]any {
	properties, required := buildSchemaFields(fields)
	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func buildSchemaFields(fields []PluginFieldInput) (map[string]any, []string) {
	properties := make(map[string]any, len(fields))
	required := make([]string, 0, len(fields))
	for _, field := range fields {
		properties[field.Name] = buildFieldSchema(field)
		if field.Required {
			required = append(required, field.Name)
		}
	}
	return properties, required
}

func buildFieldSchema(field PluginFieldInput) map[string]any {
	schema := map[string]any{
		"type": field.Type,
	}
	if field.Description != "" {
		schema["description"] = field.Description
	}
	if len(field.EnumValues) > 0 {
		schema["enum"] = field.EnumValues
	}
	if field.DefaultValue != nil {
		schema["default"] = field.DefaultValue
	}
	switch field.Type {
	case "object":
		children := buildFieldObjectSchema(field.Children)
		for key, value := range children {
			schema[key] = value
		}
	case "array":
		if field.Items != nil {
			schema["items"] = buildFieldSchema(*field.Items)
		}
	}
	return schema
}

func buildCombinedInputSchema(queryFields []PluginFieldInput, headerFields []PluginFieldInput, bodyFields []PluginFieldInput) (string, error) {
	properties := make(map[string]any)
	if len(queryFields) > 0 {
		properties["query"] = buildObjectSchema(queryFields)
	}
	if len(headerFields) > 0 {
		properties["header"] = buildObjectSchema(headerFields)
	}
	if len(bodyFields) > 0 {
		properties["body"] = buildObjectSchema(bodyFields)
	}
	return marshalJSONString(map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	})
}

func normalizeOutputSchema(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return marshalJSONString(map[string]any{
			"$schema":              "https://json-schema.org/draft/2020-12/schema",
			"type":                 "object",
			"additionalProperties": true,
		})
	}
	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return "", errors.New("output_schema_json must be valid json")
	}
	return marshalJSONString(payload)
}

func decodeFieldList(raw string) []PluginFieldInput {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []PluginFieldInput{}
	}
	var result []PluginFieldInput
	if err := json.Unmarshal([]byte(trimmed), &result); err != nil {
		return []PluginFieldInput{}
	}
	return result
}

func validateHTTPURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return errors.New("request_url must be a valid url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("request_url must start with http or https")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return errors.New("request_url must be a valid url")
	}
	return nil
}

func normalizePluginSource(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case pluginSourceCustom:
		return pluginSourceCustom
	case pluginSourceBuiltin:
		return pluginSourceBuiltin
	default:
		return ""
	}
}

func normalizeRuntimeType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case pluginRuntimeAPI:
		return pluginRuntimeAPI
	case pluginRuntimeMCP:
		return pluginRuntimeMCP
	case pluginRuntimeCode:
		return pluginRuntimeCode
	default:
		return ""
	}
}

func normalizeToolResponseMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case toolResponseStreaming:
		return toolResponseStreaming
	case toolResponseNonStreaming:
		return toolResponseNonStreaming
	default:
		return ""
	}
}

func normalizeAPIRequestType(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case apiMethodGET:
		return apiMethodGET
	case apiMethodPOST:
		return apiMethodPOST
	default:
		return ""
	}
}

func normalizeCodeLanguage(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case codeLanguagePython:
		return codeLanguagePython
	case codeLanguageJavaScript:
		return codeLanguageJavaScript
	default:
		return ""
	}
}

func normalizeMCPProtocol(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case mcpProtocolSSE:
		return mcpProtocolSSE
	case "streamablehttp":
		return mcpProtocolStreamableHTTP
	default:
		return ""
	}
}

func normalizeOptionalFilter(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func marshalJSONString(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func normalizeFieldDefaultValue(value any, fieldType string) (any, error) {
	switch fieldType {
	case "string":
		if text, ok := value.(string); ok {
			return text, nil
		}
		return fmt.Sprint(value), nil
	case "number":
		number, err := parseFloatDefaultValue(value)
		if err != nil {
			return nil, errors.New("default_value must match field type")
		}
		return number, nil
	case "integer":
		integer, err := parseIntegerDefaultValue(value)
		if err != nil {
			return nil, errors.New("default_value must match field type")
		}
		return integer, nil
	case "boolean":
		booleanValue, err := parseBooleanDefaultValue(value)
		if err != nil {
			return nil, errors.New("default_value must match field type")
		}
		return booleanValue, nil
	case "object":
		objectValue, err := parseStructuredDefaultValue(value, false)
		if err != nil {
			return nil, errors.New("default_value must be a json object")
		}
		return objectValue, nil
	case "array":
		arrayValue, err := parseStructuredDefaultValue(value, true)
		if err != nil {
			return nil, errors.New("default_value must be a json array")
		}
		return arrayValue, nil
	default:
		return nil, errors.New("default_value must match field type")
	}
}

func parseFloatDefaultValue(value any) (float64, error) {
	switch typed := value.(type) {
	case float64:
		return typed, nil
	case float32:
		return float64(typed), nil
	case int:
		return float64(typed), nil
	case int8:
		return float64(typed), nil
	case int16:
		return float64(typed), nil
	case int32:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case uint:
		return float64(typed), nil
	case uint8:
		return float64(typed), nil
	case uint16:
		return float64(typed), nil
	case uint32:
		return float64(typed), nil
	case uint64:
		return float64(typed), nil
	case json.Number:
		return typed.Float64()
	case string:
		return strconv.ParseFloat(strings.TrimSpace(typed), 64)
	default:
		return 0, errors.New("invalid number")
	}
}

func parseIntegerDefaultValue(value any) (int64, error) {
	switch typed := value.(type) {
	case float64:
		if typed != float64(int64(typed)) {
			return 0, errors.New("invalid integer")
		}
		return int64(typed), nil
	case float32:
		if typed != float32(int64(typed)) {
			return 0, errors.New("invalid integer")
		}
		return int64(typed), nil
	case int:
		return int64(typed), nil
	case int8:
		return int64(typed), nil
	case int16:
		return int64(typed), nil
	case int32:
		return int64(typed), nil
	case int64:
		return typed, nil
	case uint:
		return int64(typed), nil
	case uint8:
		return int64(typed), nil
	case uint16:
		return int64(typed), nil
	case uint32:
		return int64(typed), nil
	case uint64:
		return int64(typed), nil
	case json.Number:
		return typed.Int64()
	case string:
		return strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
	default:
		return 0, errors.New("invalid integer")
	}
}

func parseBooleanDefaultValue(value any) (bool, error) {
	switch typed := value.(type) {
	case bool:
		return typed, nil
	case string:
		return strconv.ParseBool(strings.TrimSpace(typed))
	default:
		return false, errors.New("invalid boolean")
	}
}

func parseStructuredDefaultValue(value any, expectArray bool) (any, error) {
	var parsed any
	switch typed := value.(type) {
	case string:
		if err := json.Unmarshal([]byte(strings.TrimSpace(typed)), &parsed); err != nil {
			return nil, err
		}
	default:
		parsed = typed
	}

	if expectArray {
		if arrayValue, ok := parsed.([]any); ok {
			return arrayValue, nil
		}
		return nil, errors.New("invalid array")
	}
	if objectValue, ok := parsed.(map[string]any); ok {
		return objectValue, nil
	}
	return nil, errors.New("invalid object")
}
