package plugin

import (
	"strings"

	"openIntern/internal/dao"
	"openIntern/internal/util"
)

// ResolveEnabledRuntimeToolNamesByIDs 将启用态 tool_id 列表映射为工具名，保持输入顺序。
func (s *PluginService) ResolveEnabledRuntimeToolNamesByIDs(toolIDs []string) ([]string, error) {
	return []string{}, nil
}

func (s *PluginService) ResolveEnabledRuntimeToolNamesByIDsForUser(userID string, toolIDs []string) ([]string, error) {
	toolIDs = util.NormalizeUniqueStringList(toolIDs)
	if len(toolIDs) == 0 {
		return []string{}, nil
	}

	toolNameByID, err := s.loadEnabledRuntimeToolNameMap(userID, toolIDs)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(toolIDs))
	for _, toolID := range toolIDs {
		toolName := strings.TrimSpace(toolNameByID[toolID])
		if toolName == "" {
			continue
		}
		names = append(names, toolName)
	}
	return util.NormalizeUniqueStringList(names), nil
}

func (s *PluginService) loadEnabledRuntimeToolNameMap(userID string, toolIDs []string) (map[string]string, error) {
	toolRows, err := dao.Plugin.ListEnabledToolsByIDs(userID, toolIDs)
	if err != nil {
		return nil, err
	}

	toolNameByID := make(map[string]string, len(toolRows))
	for _, row := range toolRows {
		toolNameByID[row.ToolID] = strings.TrimSpace(row.ToolName)
	}
	return toolNameByID, nil
}
