package plugin

import (
	"context"
	"fmt"

	"openIntern/internal/dao"
	"openIntern/internal/util"

	einoTool "github.com/cloudwego/eino/components/tool"
)

func (s *PluginService) BuildRuntimeAPITools(ctx context.Context, toolIDs []string) ([]einoTool.BaseTool, error) {
	toolIDs = util.NormalizeUniqueStringList(toolIDs)
	if len(toolIDs) == 0 {
		return nil, nil
	}
	return s.buildRuntimeAPITools(ctx, toolIDs)
}

// BuildAllRuntimeAPITools 构建全部启用态 api 插件工具，供动态工具池预装使用。
func (s *PluginService) BuildAllRuntimeAPITools(ctx context.Context) ([]einoTool.BaseTool, error) {
	return s.buildRuntimeAPITools(ctx, nil)
}

func (s *PluginService) buildRuntimeAPITools(ctx context.Context, toolIDs []string) ([]einoTool.BaseTool, error) {
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, nil
	}
	toolRows, err := dao.Plugin.ListRuntimeTools(userID, pluginRuntimeAPI, pluginStatusEnabled, toolIDs)
	if err != nil {
		return nil, err
	}
	if len(toolRows) == 0 {
		return nil, nil
	}

	runtimeTools := make([]einoTool.BaseTool, 0, len(toolRows))
	for _, row := range toolRows {
		runtimeTool, err := NewAPIPluginTool(row)
		if err != nil {
			return nil, fmt.Errorf("build api plugin tool %s failed: %w", row.ToolID, err)
		}
		runtimeTools = append(runtimeTools, runtimeTool)
	}
	return runtimeTools, nil
}
