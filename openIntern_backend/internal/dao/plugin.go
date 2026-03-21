package dao

import (
	"errors"
	"strings"
	"time"

	"openIntern/internal/database"
	"openIntern/internal/models"
	"openIntern/internal/util"

	"gorm.io/gorm"
)

var ErrPluginNotFound = errors.New("plugin not found")

type PluginListFilter struct {
	Source      string
	RuntimeType string
	Status      string
	Keyword     string
}

// EnabledRuntimeToolRecord 表示可运行工具及其运行时类型。
type EnabledRuntimeToolRecord struct {
	ToolID      string `gorm:"column:tool_id"`
	RuntimeType string `gorm:"column:runtime_type"`
}

type PluginDAO struct{}

var Plugin = new(PluginDAO)

func (d *PluginDAO) Create(plugin *models.Plugin, tools []models.Tool) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(plugin).Error; err != nil {
			return err
		}
		if len(tools) == 0 {
			return nil
		}
		return tx.Create(&tools).Error
	})
}

func (d *PluginDAO) Update(pluginID string, plugin *models.Plugin, tools []models.Tool) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Plugin{}).
			Where("plugin_id = ?", pluginID).
			Updates(map[string]any{
				"name":         plugin.Name,
				"description":  plugin.Description,
				"icon":         plugin.Icon,
				"source":       plugin.Source,
				"runtime_type": plugin.RuntimeType,
				"status":       plugin.Status,
				"mcp_url":      plugin.MCPURL,
				"mcp_protocol": plugin.MCPProtocol,
				"last_sync_at": plugin.LastSyncAt,
			}).Error; err != nil {
			return err
		}

		var existingTools []models.Tool
		if err := tx.Where("plugin_id = ?", pluginID).Find(&existingTools).Error; err != nil {
			return err
		}

		toolsToUpdate, toolsToCreate, removedToolIDs := diffPluginTools(existingTools, tools)
		if len(removedToolIDs) > 0 {
			if err := tx.Where("tool_id IN ?", removedToolIDs).Delete(&models.PluginDefault{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Where("tool_id IN ?", removedToolIDs).Delete(&models.Tool{}).Error; err != nil {
				return err
			}
		}

		for _, tool := range toolsToUpdate {
			if err := tx.Model(&models.Tool{}).
				Where("tool_id = ? AND plugin_id = ?", tool.ToolID, pluginID).
				Updates(buildToolUpdateMap(tool)).Error; err != nil {
				return err
			}
		}

		if len(toolsToCreate) == 0 {
			return nil
		}
		return tx.Create(&toolsToCreate).Error
	})
}

func (d *PluginDAO) GetByPluginID(pluginID string) (*models.Plugin, error) {
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return nil, ErrPluginNotFound
	}

	var item models.Plugin
	if err := database.DB.Where("plugin_id = ?", pluginID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPluginNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (d *PluginDAO) List(page, pageSize int, filter PluginListFilter) ([]models.Plugin, int64, error) {
	db := applyPluginFilters(database.DB.Model(&models.Plugin{}), filter)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []models.Plugin
	offset := (page - 1) * pageSize
	if err := db.Order("updated_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (d *PluginDAO) Delete(pluginID string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var toolIDs []string
		if err := tx.Model(&models.Tool{}).Where("plugin_id = ?", pluginID).Pluck("tool_id", &toolIDs).Error; err != nil {
			return err
		}
		if len(toolIDs) > 0 {
			if err := tx.Where("tool_id IN ?", toolIDs).Delete(&models.PluginDefault{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("plugin_id = ?", pluginID).Delete(&models.Tool{}).Error; err != nil {
			return err
		}
		return tx.Where("plugin_id = ?", pluginID).Delete(&models.Plugin{}).Error
	})
}

func (d *PluginDAO) UpdateStatus(pluginID string, status string) error {
	return database.DB.Model(&models.Plugin{}).Where("plugin_id = ?", pluginID).Update("status", status).Error
}

// UpdateLastSyncAt 更新插件最近同步时间。
func (d *PluginDAO) UpdateLastSyncAt(pluginID string, syncedAt time.Time) error {
	return database.DB.Model(&models.Plugin{}).Where("plugin_id = ?", pluginID).Update("last_sync_at", &syncedAt).Error
}

func (d *PluginDAO) ListEnabled() ([]models.Plugin, error) {
	var plugins []models.Plugin
	if err := database.DB.
		Where("status = ?", "enabled").
		Order("updated_at DESC").
		Find(&plugins).Error; err != nil {
		return nil, err
	}
	return plugins, nil
}

// ListAll 返回全部插件记录。
func (d *PluginDAO) ListAll() ([]models.Plugin, error) {
	var plugins []models.Plugin
	if err := database.DB.
		Order("updated_at DESC").
		Find(&plugins).Error; err != nil {
		return nil, err
	}
	return plugins, nil
}

func (d *PluginDAO) ListByIDsAndRuntimeStatus(pluginIDs []string, runtimeType, status string) ([]models.Plugin, error) {
	if len(pluginIDs) == 0 {
		return nil, nil
	}

	var plugins []models.Plugin
	if err := database.DB.
		Where("plugin_id IN ? AND runtime_type = ? AND status = ?", pluginIDs, runtimeType, status).
		Find(&plugins).Error; err != nil {
		return nil, err
	}
	return plugins, nil
}

// ListByRuntimeStatus 返回指定运行时和状态的全部插件。
func (d *PluginDAO) ListByRuntimeStatus(runtimeType, status string) ([]models.Plugin, error) {
	if database.DB == nil {
		return nil, nil
	}

	var plugins []models.Plugin
	if err := database.DB.
		Where("runtime_type = ? AND status = ?", runtimeType, status).
		Find(&plugins).Error; err != nil {
		return nil, err
	}
	return plugins, nil
}

func (d *PluginDAO) ListRuntimeTools(runtimeType, status string, toolIDs []string) ([]models.Tool, error) {
	query := database.DB.Model(&models.Tool{}).
		Joins("JOIN plugin ON plugin.plugin_id = tool.plugin_id").
		Where("plugin.runtime_type = ? AND plugin.status = ? AND tool.enabled = ?", runtimeType, status, true)
	if len(toolIDs) > 0 {
		query = query.Where("tool.tool_id IN ?", toolIDs)
	}

	var tools []models.Tool
	if err := query.Order("tool.tool_name ASC").Find(&tools).Error; err != nil {
		return nil, err
	}
	return tools, nil
}

// ListEnabledRuntimeToolRecords 根据 tool_id 列表过滤可运行工具并返回运行时类型。
func (d *PluginDAO) ListEnabledRuntimeToolRecords(toolIDs []string, runtimeTypes []string) ([]EnabledRuntimeToolRecord, error) {
	toolIDs = util.NormalizeUniqueStringList(toolIDs)
	if len(toolIDs) == 0 {
		return []EnabledRuntimeToolRecord{}, nil
	}

	query := database.DB.
		Table("tool").
		Select("tool.tool_id AS tool_id, plugin.runtime_type AS runtime_type").
		Joins("JOIN plugin ON plugin.plugin_id = tool.plugin_id").
		Where("tool.tool_id IN ? AND tool.enabled = ? AND plugin.status = ?", toolIDs, true, "enabled")
	if len(runtimeTypes) > 0 {
		query = query.Where("plugin.runtime_type IN ?", runtimeTypes)
	}

	var rows []EnabledRuntimeToolRecord
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ListEnabledToolsByIDs 返回指定 tool_id 对应的启用态工具定义，用于运行时可见性映射。
func (d *PluginDAO) ListEnabledToolsByIDs(toolIDs []string) ([]models.Tool, error) {
	toolIDs = util.NormalizeUniqueStringList(toolIDs)
	if len(toolIDs) == 0 {
		return []models.Tool{}, nil
	}

	var tools []models.Tool
	if err := database.DB.
		Table("tool").
		Select("tool.*").
		Joins("JOIN plugin ON plugin.plugin_id = tool.plugin_id").
		Where("tool.tool_id IN ? AND tool.enabled = ? AND plugin.status = ?", toolIDs, true, "enabled").
		Find(&tools).Error; err != nil {
		return nil, err
	}
	return tools, nil
}

func (d *PluginDAO) LoadToolMap(pluginIDs []string) (map[string][]models.Tool, error) {
	result := make(map[string][]models.Tool, len(pluginIDs))
	if len(pluginIDs) == 0 {
		return result, nil
	}

	var tools []models.Tool
	if err := database.DB.
		Where("plugin_id IN ?", pluginIDs).
		Order("updated_at DESC").
		Find(&tools).Error; err != nil {
		return nil, err
	}

	for _, tool := range tools {
		result[tool.PluginID] = append(result[tool.PluginID], tool)
	}
	return result, nil
}

func (d *PluginDAO) ListToolsByPluginID(pluginID string) ([]models.Tool, error) {
	var tools []models.Tool
	if err := database.DB.Where("plugin_id = ?", pluginID).Find(&tools).Error; err != nil {
		return nil, err
	}
	return tools, nil
}

func (d *PluginDAO) ReplaceToolsAndUpdateSyncTime(pluginID string, syncedTools []models.Tool, syncedAt time.Time) error {
	existingTools, err := d.ListToolsByPluginID(pluginID)
	if err != nil {
		return err
	}

	return database.DB.Transaction(func(tx *gorm.DB) error {
		toolsToUpdate, toolsToCreate, removedToolIDs := diffPluginTools(existingTools, syncedTools)
		if len(removedToolIDs) > 0 {
			if err := tx.Where("tool_id IN ?", removedToolIDs).Delete(&models.PluginDefault{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Where("tool_id IN ?", removedToolIDs).Delete(&models.Tool{}).Error; err != nil {
				return err
			}
		}
		for _, tool := range toolsToUpdate {
			if err := tx.Model(&models.Tool{}).
				Where("tool_id = ? AND plugin_id = ?", tool.ToolID, pluginID).
				Updates(buildToolUpdateMap(tool)).Error; err != nil {
				return err
			}
		}
		if len(toolsToCreate) > 0 {
			if err := tx.Create(&toolsToCreate).Error; err != nil {
				return err
			}
		}
		return tx.Model(&models.Plugin{}).
			Where("plugin_id = ?", pluginID).
			Update("last_sync_at", &syncedAt).Error
	})
}

func applyPluginFilters(db *gorm.DB, filter PluginListFilter) *gorm.DB {
	if filter.Source != "" {
		db = db.Where("source = ?", filter.Source)
	}
	if filter.RuntimeType != "" {
		db = db.Where("runtime_type = ?", filter.RuntimeType)
	}
	if filter.Status != "" {
		db = db.Where("status = ?", filter.Status)
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		pattern := "%" + keyword + "%"
		db = db.Where("(name LIKE ? OR description LIKE ?)", pattern, pattern)
	}
	return db
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
