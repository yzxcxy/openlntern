package services

import (
	"context"
	"fmt"

	"openIntern/internal/dao"
	toolsvc "openIntern/internal/services/tools"

	einoTool "github.com/cloudwego/eino/components/tool"
)

func (s *PluginService) BuildRuntimeCodeTools(ctx context.Context, toolIDs []string) ([]einoTool.BaseTool, error) {
	_ = ctx

	toolIDs = normalizeUniqueStringList(toolIDs)
	if len(toolIDs) == 0 {
		return nil, nil
	}

	toolRows, err := dao.Plugin.ListRuntimeTools(pluginRuntimeCode, pluginStatusEnabled, toolIDs)
	if err != nil {
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
