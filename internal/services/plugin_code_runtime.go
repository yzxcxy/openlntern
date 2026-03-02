package services

import (
	"context"
	"fmt"

	"openIntern/internal/database"
	"openIntern/internal/models"
	toolsvc "openIntern/internal/services/tools"

	einoTool "github.com/cloudwego/eino/components/tool"
)

func (s *PluginService) BuildRuntimeCodeTools(ctx context.Context, toolIDs []string) ([]einoTool.BaseTool, error) {
	_ = ctx

	toolIDs = normalizeUniqueStringList(toolIDs)
	if len(toolIDs) == 0 {
		return nil, nil
	}

	db := database.DB.Model(&models.Tool{}).
		Joins("JOIN plugin ON plugin.plugin_id = tool.plugin_id").
		Where("plugin.runtime_type = ? AND plugin.status = ? AND tool.enabled = ?", pluginRuntimeCode, pluginStatusEnabled, true)
	if len(toolIDs) > 0 {
		db = db.Where("tool.tool_id IN ?", toolIDs)
	}

	var toolRows []models.Tool
	if err := db.Order("tool.tool_name ASC").Find(&toolRows).Error; err != nil {
		return nil, err
	}
	if len(toolRows) == 0 {
		return nil, nil
	}

	runtimeTools := make([]einoTool.BaseTool, 0, len(toolRows))
	for _, row := range toolRows {
		runtimeTool, err := toolsvc.NewCodePluginTool(row)
		if err != nil {
			return nil, fmt.Errorf("build code plugin tool %s failed: %w", row.ToolID, err)
		}
		runtimeTools = append(runtimeTools, runtimeTool)
	}
	return runtimeTools, nil
}
