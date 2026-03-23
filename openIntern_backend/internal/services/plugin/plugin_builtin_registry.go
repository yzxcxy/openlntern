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
func ensureBuiltinPlugins(manifestPath string) error {
	definitions, err := loadBuiltinPluginManifest(manifestPath)
	if err != nil {
		return err
	}
	for _, definition := range definitions {
		if err := upsertBuiltinPlugin(definition); err != nil {
			return err
		}
	}
	return nil
}

func upsertBuiltinPlugin(definition builtinPluginDefinition) error {
	plugin := definition.plugin
	plugin.Name = strings.TrimSpace(plugin.Name)
	if plugin.Name == "" {
		return errors.New("builtin plugin name is required")
	}

	existing, err := dao.Plugin.GetByPluginID(plugin.PluginID)
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
	if err := dao.Plugin.Update(plugin.PluginID, &plugin, definition.tools); err != nil {
		return fmt.Errorf("update builtin plugin %s failed: %w", plugin.PluginID, err)
	}
	return nil
}
