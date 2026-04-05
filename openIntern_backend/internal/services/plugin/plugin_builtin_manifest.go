package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"openIntern/internal/models"
	"openIntern/internal/util"

	"gopkg.in/yaml.v3"
)

const defaultBuiltinManifestPath = "builtin_plugins.yaml"

type builtinPluginManifestFile struct {
	Plugins []builtinPluginManifestPlugin `yaml:"plugins"`
}

type builtinPluginManifestPlugin struct {
	PluginID    string                      `yaml:"plugin_id"`
	Name        string                      `yaml:"name"`
	Description string                      `yaml:"description"`
	Icon        string                      `yaml:"icon"`
	Source      string                      `yaml:"source"`
	RuntimeType string                      `yaml:"runtime_type"`
	Status      string                      `yaml:"status"`
	Tools       []builtinPluginManifestTool `yaml:"tools"`
}

type builtinPluginManifestTool struct {
	ToolID           string `yaml:"tool_id"`
	ToolName         string `yaml:"tool_name"`
	Description      string `yaml:"description"`
	InputSchemaJSON  string `yaml:"input_schema_json"`
	OutputSchemaJSON string `yaml:"output_schema_json"`
	ToolResponseMode string `yaml:"tool_response_mode"`
	Enabled          *bool  `yaml:"enabled"`
	TimeoutMS        int    `yaml:"timeout_ms"`
}

// loadBuiltinPluginManifest reads builtin plugin metadata from YAML so future additions can be made without editing code.
func loadBuiltinPluginManifest(manifestPath string) ([]builtinPluginDefinition, error) {
	manifestPath = strings.TrimSpace(manifestPath)
	if manifestPath == "" {
		manifestPath = defaultBuiltinManifestPath
	}

	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read builtin plugin manifest %s failed: %w", manifestPath, err)
	}

	var manifest builtinPluginManifestFile
	if err := yaml.Unmarshal(raw, &manifest); err != nil {
		return nil, fmt.Errorf("parse builtin plugin manifest %s failed: %w", manifestPath, err)
	}
	if len(manifest.Plugins) == 0 {
		return nil, fmt.Errorf("builtin plugin manifest %s has no plugins", manifestPath)
	}

	definitions := make([]builtinPluginDefinition, 0, len(manifest.Plugins))
	seenPluginIDs := make(map[string]struct{}, len(manifest.Plugins))
	for _, item := range manifest.Plugins {
		definition, err := buildBuiltinPluginDefinition(item)
		if err != nil {
			return nil, err
		}
		if _, exists := seenPluginIDs[definition.plugin.PluginID]; exists {
			return nil, fmt.Errorf("duplicate builtin plugin_id: %s", definition.plugin.PluginID)
		}
		seenPluginIDs[definition.plugin.PluginID] = struct{}{}
		definitions = append(definitions, definition)
	}
	return definitions, nil
}

func buildBuiltinPluginDefinition(item builtinPluginManifestPlugin) (builtinPluginDefinition, error) {
	pluginID := strings.TrimSpace(item.PluginID)
	if pluginID == "" {
		return builtinPluginDefinition{}, errors.New("builtin plugin_id is required")
	}
	name := strings.TrimSpace(item.Name)
	if name == "" {
		return builtinPluginDefinition{}, fmt.Errorf("builtin plugin name is required: %s", pluginID)
	}

	source := normalizePluginSource(item.Source)
	if source == "" {
		source = pluginSourceBuiltin
	}
	if source != pluginSourceBuiltin {
		return builtinPluginDefinition{}, fmt.Errorf("builtin plugin source must be builtin: %s", pluginID)
	}

	runtimeType := normalizeRuntimeType(item.RuntimeType)
	if runtimeType == "" {
		runtimeType = pluginRuntimeBuiltin
	}
	if runtimeType != pluginRuntimeBuiltin {
		return builtinPluginDefinition{}, fmt.Errorf("builtin plugin runtime_type must be builtin: %s", pluginID)
	}

	status := strings.ToLower(strings.TrimSpace(item.Status))
	if status == "" {
		status = pluginStatusEnabled
	}
	if status != pluginStatusEnabled && status != pluginStatusDisabled {
		return builtinPluginDefinition{}, fmt.Errorf("unsupported builtin plugin status: %s", pluginID)
	}

	plugin := models.Plugin{
		PluginID:    pluginID,
		Name:        name,
		Description: strings.TrimSpace(item.Description),
		Icon:        normalizePluginIconForStorage(strings.TrimSpace(item.Icon)),
		Source:      source,
		RuntimeType: runtimeType,
		Status:      status,
	}

	if len(item.Tools) == 0 {
		return builtinPluginDefinition{}, fmt.Errorf("builtin plugin has no tools: %s", pluginID)
	}

	tools := make([]models.Tool, 0, len(item.Tools))
	seenToolIDs := make(map[string]struct{}, len(item.Tools))
	seenToolNames := make(map[string]struct{}, len(item.Tools))
	for _, toolItem := range item.Tools {
		tool, err := buildBuiltinManifestTool(pluginID, toolItem)
		if err != nil {
			return builtinPluginDefinition{}, err
		}
		if _, exists := seenToolIDs[tool.ToolID]; exists {
			return builtinPluginDefinition{}, fmt.Errorf("duplicate builtin tool_id: %s", tool.ToolID)
		}
		if _, exists := seenToolNames[tool.ToolName]; exists {
			return builtinPluginDefinition{}, fmt.Errorf("duplicate builtin tool_name in plugin %s: %s", pluginID, tool.ToolName)
		}
		seenToolIDs[tool.ToolID] = struct{}{}
		seenToolNames[tool.ToolName] = struct{}{}
		tools = append(tools, tool)
	}
	return builtinPluginDefinition{plugin: plugin, tools: tools}, nil
}

func buildBuiltinManifestTool(pluginID string, item builtinPluginManifestTool) (models.Tool, error) {
	toolID := strings.TrimSpace(item.ToolID)
	if toolID == "" {
		return models.Tool{}, fmt.Errorf("builtin tool_id is required for plugin %s", pluginID)
	}
	toolName := strings.TrimSpace(item.ToolName)
	if toolName == "" {
		return models.Tool{}, fmt.Errorf("builtin tool_name is required for plugin %s", pluginID)
	}
	if !util.IsModelSafeToolName(toolName) {
		return models.Tool{}, fmt.Errorf("builtin tool_name must match ^[a-zA-Z0-9_-]+$: %s", toolName)
	}

	inputSchemaJSON, err := normalizeBuiltinSchemaJSON(item.InputSchemaJSON, "input_schema_json")
	if err != nil {
		return models.Tool{}, fmt.Errorf("builtin tool %s %w", toolName, err)
	}
	outputSchemaJSON, err := normalizeBuiltinSchemaJSON(item.OutputSchemaJSON, "output_schema_json")
	if err != nil {
		return models.Tool{}, fmt.Errorf("builtin tool %s %w", toolName, err)
	}

	responseMode := normalizeToolResponseMode(item.ToolResponseMode)
	if responseMode == "" {
		responseMode = toolResponseNonStreaming
	}

	enabled := true
	if item.Enabled != nil {
		enabled = *item.Enabled
	}

	timeoutMS := item.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = defaultPluginTimeoutMS
	}

	return models.Tool{
		ToolID:           toolID,
		PluginID:         pluginID,
		ToolName:         toolName,
		Description:      strings.TrimSpace(item.Description),
		InputSchemaJSON:  inputSchemaJSON,
		OutputSchemaJSON: outputSchemaJSON,
		ToolResponseMode: responseMode,
		Enabled:          enabled,
		TimeoutMS:        timeoutMS,
	}, nil
}

func normalizeBuiltinSchemaJSON(raw string, fieldName string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%s is required", fieldName)
	}
	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return "", fmt.Errorf("%s must be valid json", fieldName)
	}
	return marshalJSONString(payload)
}
