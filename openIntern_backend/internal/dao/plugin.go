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
	UserID      string
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

// EnabledRuntimeToolSearchRow 表示本地 tool_search 所需的最小联合视图。
type EnabledRuntimeToolSearchRow struct {
	ToolID            string `gorm:"column:tool_id"`
	ToolName          string `gorm:"column:tool_name"`
	Description       string `gorm:"column:description"`
	RuntimeType       string `gorm:"column:runtime_type"`
	PluginName        string `gorm:"column:plugin_name"`
	PluginDescription string `gorm:"column:plugin_description"`
}

type PluginDAO struct{}

var Plugin = new(PluginDAO)

func (d *PluginDAO) Create(plugin *models.Plugin, tools []models.Tool) error {
	userID := strings.TrimSpace(plugin.UserID)
	if userID == "" {
		return ErrPluginNotFound
	}
	for i := range tools {
		tools[i].UserID = userID
	}
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

func (d *PluginDAO) Update(userID string, pluginID string, plugin *models.Plugin, tools []models.Tool) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrPluginNotFound
	}
	for i := range tools {
		tools[i].UserID = userID
	}
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Plugin{}).
			Where("user_id = ? AND plugin_id = ?", userID, pluginID).
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
		if err := tx.Where("user_id = ? AND plugin_id = ?", userID, pluginID).Find(&existingTools).Error; err != nil {
			return err
		}

		toolsToUpdate, toolsToCreate, removedToolIDs := diffPluginTools(existingTools, tools)
		if len(removedToolIDs) > 0 {
			if err := tx.Where("user_id = ? AND tool_id IN ?", userID, removedToolIDs).Delete(&models.PluginDefault{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Where("user_id = ? AND tool_id IN ?", userID, removedToolIDs).Delete(&models.Tool{}).Error; err != nil {
				return err
			}
		}

		for _, tool := range toolsToUpdate {
			if err := tx.Model(&models.Tool{}).
				Where("user_id = ? AND tool_id = ? AND plugin_id = ?", userID, tool.ToolID, pluginID).
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

func (d *PluginDAO) GetByUserIDAndPluginID(userID string, pluginID string) (*models.Plugin, error) {
	userID = strings.TrimSpace(userID)
	pluginID = strings.TrimSpace(pluginID)
	if userID == "" || pluginID == "" {
		return nil, ErrPluginNotFound
	}

	var item models.Plugin
	if err := database.DB.Where("user_id = ? AND plugin_id = ?", userID, pluginID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPluginNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (d *PluginDAO) GetBuiltinByUserIDAndName(userID string, name string) (*models.Plugin, error) {
	userID = strings.TrimSpace(userID)
	name = strings.TrimSpace(name)
	if userID == "" || name == "" {
		return nil, ErrPluginNotFound
	}

	var item models.Plugin
	if err := database.DB.Where("user_id = ? AND source = ? AND name = ?", userID, "builtin", name).First(&item).Error; err != nil {
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

func (d *PluginDAO) Delete(userID string, pluginID string) error {
	userID = strings.TrimSpace(userID)
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var toolIDs []string
		if err := tx.Model(&models.Tool{}).Where("user_id = ? AND plugin_id = ?", userID, pluginID).Pluck("tool_id", &toolIDs).Error; err != nil {
			return err
		}
		if len(toolIDs) > 0 {
			if err := tx.Where("user_id = ? AND tool_id IN ?", userID, toolIDs).Delete(&models.PluginDefault{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("user_id = ? AND plugin_id = ?", userID, pluginID).Delete(&models.Tool{}).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ? AND plugin_id = ?", userID, pluginID).Delete(&models.Plugin{}).Error
	})
}

func (d *PluginDAO) UpdateStatus(userID string, pluginID string, status string) error {
	return database.DB.Model(&models.Plugin{}).Where("user_id = ? AND plugin_id = ?", strings.TrimSpace(userID), pluginID).Update("status", status).Error
}

// UpdateLastSyncAt 更新插件最近同步时间。
func (d *PluginDAO) UpdateLastSyncAt(userID string, pluginID string, syncedAt time.Time) error {
	return database.DB.Model(&models.Plugin{}).Where("user_id = ? AND plugin_id = ?", strings.TrimSpace(userID), pluginID).Update("last_sync_at", &syncedAt).Error
}

func (d *PluginDAO) ListEnabled(userID string) ([]models.Plugin, error) {
	var plugins []models.Plugin
	if err := database.DB.
		Where("user_id = ? AND status = ?", strings.TrimSpace(userID), "enabled").
		Order("updated_at DESC").
		Find(&plugins).Error; err != nil {
		return nil, err
	}
	return plugins, nil
}

// ListAll 返回全部插件记录。
func (d *PluginDAO) ListAll(userID string) ([]models.Plugin, error) {
	var plugins []models.Plugin
	if err := database.DB.
		Where("user_id = ?", strings.TrimSpace(userID)).
		Order("updated_at DESC").
		Find(&plugins).Error; err != nil {
		return nil, err
	}
	return plugins, nil
}

func (d *PluginDAO) ListByIDsAndRuntimeStatus(userID string, pluginIDs []string, runtimeType, status string) ([]models.Plugin, error) {
	if len(pluginIDs) == 0 {
		return nil, nil
	}

	var plugins []models.Plugin
	if err := database.DB.
		Where("user_id = ? AND plugin_id IN ? AND runtime_type = ? AND status = ?", strings.TrimSpace(userID), pluginIDs, runtimeType, status).
		Find(&plugins).Error; err != nil {
		return nil, err
	}
	return plugins, nil
}

// ListByRuntimeStatus 返回指定运行时和状态的全部插件。
func (d *PluginDAO) ListByRuntimeStatus(userID, runtimeType, status string) ([]models.Plugin, error) {
	if database.DB == nil {
		return nil, nil
	}

	var plugins []models.Plugin
	if err := database.DB.
		Where("user_id = ? AND runtime_type = ? AND status = ?", strings.TrimSpace(userID), runtimeType, status).
		Find(&plugins).Error; err != nil {
		return nil, err
	}
	return plugins, nil
}

// ListByRuntimeStatusAll 返回全部用户下指定运行时和状态的插件。
func (d *PluginDAO) ListByRuntimeStatusAll(runtimeType, status string) ([]models.Plugin, error) {
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

func (d *PluginDAO) ListRuntimeTools(userID, runtimeType, status string, toolIDs []string) ([]models.Tool, error) {
	query := database.DB.Model(&models.Tool{}).
		Joins("JOIN plugin ON plugin.plugin_id = tool.plugin_id AND plugin.user_id = tool.user_id").
		Where("tool.user_id = ? AND plugin.runtime_type = ? AND plugin.status = ? AND tool.enabled = ?", strings.TrimSpace(userID), runtimeType, status, true)
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
func (d *PluginDAO) ListEnabledRuntimeToolRecords(userID string, toolIDs []string, runtimeTypes []string) ([]EnabledRuntimeToolRecord, error) {
	toolIDs = util.NormalizeUniqueStringList(toolIDs)
	if len(toolIDs) == 0 {
		return []EnabledRuntimeToolRecord{}, nil
	}

	query := database.DB.
		Table("tool").
		Select("tool.tool_id AS tool_id, plugin.runtime_type AS runtime_type").
		Joins("JOIN plugin ON plugin.plugin_id = tool.plugin_id AND plugin.user_id = tool.user_id").
		Where("tool.user_id = ? AND tool.tool_id IN ? AND tool.enabled = ? AND plugin.status = ?", strings.TrimSpace(userID), toolIDs, true, "enabled")
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
func (d *PluginDAO) ListEnabledToolsByIDs(userID string, toolIDs []string) ([]models.Tool, error) {
	toolIDs = util.NormalizeUniqueStringList(toolIDs)
	if len(toolIDs) == 0 {
		return []models.Tool{}, nil
	}

	var tools []models.Tool
	if err := database.DB.
		Table("tool").
		Select("tool.*").
		Joins("JOIN plugin ON plugin.plugin_id = tool.plugin_id AND plugin.user_id = tool.user_id").
		Where("tool.user_id = ? AND tool.tool_id IN ? AND tool.enabled = ? AND plugin.status = ?", strings.TrimSpace(userID), toolIDs, true, "enabled").
		Find(&tools).Error; err != nil {
		return nil, err
	}
	return tools, nil
}

// ListEnabledRuntimeToolSearchRows 返回本地工具搜索所需的启用态工具视图。
func (d *PluginDAO) ListEnabledRuntimeToolSearchRows(userID string, runtimeTypes []string) ([]EnabledRuntimeToolSearchRow, error) {
	query := database.DB.
		Table("tool").
		Select(
			"tool.tool_id AS tool_id, tool.tool_name AS tool_name, tool.description AS description, "+
				"plugin.runtime_type AS runtime_type, plugin.name AS plugin_name, plugin.description AS plugin_description",
		).
		Joins("JOIN plugin ON plugin.plugin_id = tool.plugin_id AND plugin.user_id = tool.user_id").
		Where("tool.user_id = ? AND tool.enabled = ? AND plugin.status = ?", strings.TrimSpace(userID), true, "enabled")
	if len(runtimeTypes) > 0 {
		query = query.Where("plugin.runtime_type IN ?", runtimeTypes)
	}

	var rows []EnabledRuntimeToolSearchRow
	if err := query.Order("tool.tool_name ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (d *PluginDAO) LoadToolMap(userID string, pluginIDs []string) (map[string][]models.Tool, error) {
	result := make(map[string][]models.Tool, len(pluginIDs))
	if len(pluginIDs) == 0 {
		return result, nil
	}

	var tools []models.Tool
	if err := database.DB.
		Where("user_id = ? AND plugin_id IN ?", strings.TrimSpace(userID), pluginIDs).
		Order("updated_at DESC").
		Find(&tools).Error; err != nil {
		return nil, err
	}

	for _, tool := range tools {
		result[tool.PluginID] = append(result[tool.PluginID], tool)
	}
	return result, nil
}

func (d *PluginDAO) ListToolsByUserIDAndPluginID(userID string, pluginID string) ([]models.Tool, error) {
	var tools []models.Tool
	if err := database.DB.Where("user_id = ? AND plugin_id = ?", strings.TrimSpace(userID), pluginID).Find(&tools).Error; err != nil {
		return nil, err
	}
	return tools, nil
}

func (d *PluginDAO) ReplaceToolsAndUpdateSyncTime(userID string, pluginID string, syncedTools []models.Tool, syncedAt time.Time) error {
	userID = strings.TrimSpace(userID)
	existingTools, err := d.ListToolsByUserIDAndPluginID(userID, pluginID)
	if err != nil {
		return err
	}
	for i := range syncedTools {
		syncedTools[i].UserID = userID
	}

	return database.DB.Transaction(func(tx *gorm.DB) error {
		toolsToUpdate, toolsToCreate, removedToolIDs := diffPluginTools(existingTools, syncedTools)
		if len(removedToolIDs) > 0 {
			if err := tx.Where("user_id = ? AND tool_id IN ?", userID, removedToolIDs).Delete(&models.PluginDefault{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Where("user_id = ? AND tool_id IN ?", userID, removedToolIDs).Delete(&models.Tool{}).Error; err != nil {
				return err
			}
		}
		for _, tool := range toolsToUpdate {
			if err := tx.Model(&models.Tool{}).
				Where("user_id = ? AND tool_id = ? AND plugin_id = ?", userID, tool.ToolID, pluginID).
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
			Where("user_id = ? AND plugin_id = ?", userID, pluginID).
			Update("last_sync_at", &syncedAt).Error
	})
}

func applyPluginFilters(db *gorm.DB, filter PluginListFilter) *gorm.DB {
	if userID := strings.TrimSpace(filter.UserID); userID != "" {
		db = db.Where("user_id = ?", userID)
	}
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
