package plugin

import (
	"errors"
	"fmt"
	"strings"

	"openIntern/internal/dao"
	"openIntern/internal/models"
)

type builtinPluginDefinition struct {
	plugin models.Plugin
	tools  []models.Tool
}

// ensureBuiltinPlugins materializes builtin tool definitions into the same plugin tables used by custom plugins.
func loadBuiltinPluginDefinitions(manifestPath string) ([]builtinPluginDefinition, error) {
	definitions, err := loadBuiltinPluginManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	return definitions, nil
}

func upsertBuiltinPlugin(userID string, definition builtinPluginDefinition) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("user_id is required")
	}
	plugin := definition.plugin
	plugin.Name = strings.TrimSpace(plugin.Name)
	if plugin.Name == "" {
		return errors.New("builtin plugin name is required")
	}
	plugin.UserID = userID
	for i := range definition.tools {
		definition.tools[i].UserID = userID
		definition.tools[i].PluginID = plugin.PluginID
	}

	existing, err := dao.Plugin.GetBuiltinByUserIDAndName(userID, plugin.Name)
	if err != nil && !errors.Is(err, dao.ErrPluginNotFound) {
		return err
	}

	if existing == nil {
		if err := dao.Plugin.Create(&plugin, definition.tools); err != nil {
			return fmt.Errorf("create builtin plugin %s failed: %w", plugin.PluginID, err)
		}
		return nil
	}

	plugin.ID = existing.ID
	plugin.CreatedAt = existing.CreatedAt
	plugin.PluginID = existing.PluginID
	for i := range definition.tools {
		definition.tools[i].PluginID = existing.PluginID
	}
	if err := dao.Plugin.Update(userID, existing.PluginID, &plugin, definition.tools); err != nil {
		return fmt.Errorf("update builtin plugin %s failed: %w", plugin.PluginID, err)
	}
	return nil
}
