package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"openIntern/internal/config"
	"openIntern/internal/dao"
	"openIntern/internal/database"
	pluginsvc "openIntern/internal/services/plugin"
)

const pluginRuntimeMCP = "mcp"

// migratorOptions 表示迁移脚本参数。
type migratorOptions struct {
	ConfigPath  string
	PluginIDs   []string
	TimeoutSecs int
}

// pluginToolStats 表示单插件工具统计信息。
type pluginToolStats struct {
	Total                int
	Enabled              int
	Disabled             int
	UniqueToolNames      int
	DuplicateNameEntries int
}

// main 读取参数并执行 MCP 插件迁移。
func main() {
	options := parseFlags()
	if err := runMigration(options); err != nil {
		log.Printf("migrate mcp tools to openviking failed: %v", err)
		os.Exit(1)
	}
}

// parseFlags 解析命令行参数。
func parseFlags() migratorOptions {
	var configPath string
	var pluginIDsArg string
	var timeoutSecs int

	flag.StringVar(&configPath, "config", "config.yaml", "config file path")
	flag.StringVar(&pluginIDsArg, "plugin_ids", "", "comma-separated plugin ids, empty means all mcp plugins")
	flag.IntVar(&timeoutSecs, "timeout", 1800, "migration timeout in seconds")
	flag.Parse()

	return migratorOptions{
		ConfigPath:  strings.TrimSpace(configPath),
		PluginIDs:   splitCommaList(pluginIDsArg),
		TimeoutSecs: timeoutSecs,
	}
}

// runMigration 执行 MCP 插件迁移流程。
func runMigration(options migratorOptions) error {
	cfg, resolvedConfigPath, err := loadConfigWithFallback(options.ConfigPath)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.MySQL.DSN) == "" {
		return fmt.Errorf("mysql.dsn is empty in config: %s", resolvedConfigPath)
	}
	if err := database.Init(cfg.MySQL.DSN); err != nil {
		return fmt.Errorf("init mysql failed: %w", err)
	}
	database.InitContextStore(cfg.Tools.OpenViking)
	if !dao.Plugin.ToolStoreConfigured() {
		return fmt.Errorf("openviking tool store is not configured")
	}

	pluginIDs, err := resolvePluginIDs(options.PluginIDs)
	if err != nil {
		return err
	}
	if len(pluginIDs) == 0 {
		log.Printf("no mcp plugins found, skip migration")
		return nil
	}

	ctx := context.Background()
	cancel := func() {}
	if options.TimeoutSecs > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(options.TimeoutSecs)*time.Second)
	}
	defer cancel()

	total := len(pluginIDs)
	success := 0
	skipped := 0
	failed := make([]string, 0)

	for _, pluginID := range pluginIDs {
		pluginRecord, getErr := dao.Plugin.GetByPluginID(pluginID)
		if errors.Is(getErr, dao.ErrPluginNotFound) {
			log.Printf("skip plugin migration plugin_id=%s reason=plugin_not_found", pluginID)
			skipped++
			continue
		}
		if getErr != nil {
			log.Printf("load plugin failed plugin_id=%s err=%v", pluginID, getErr)
			failed = append(failed, pluginID)
			continue
		}
		if !isMCPRuntime(pluginRecord.RuntimeType) {
			skipped++
			log.Printf("skip non-mcp plugin migration plugin_id=%s runtime_type=%s", pluginID, pluginRecord.RuntimeType)
			continue
		}

		stats, statErr := collectPluginToolStats(pluginID)
		if statErr != nil {
			log.Printf("collect plugin tool stats failed plugin_id=%s err=%v", pluginID, statErr)
		} else {
			log.Printf(
				"plugin tool snapshot before migration plugin_id=%s total=%d enabled=%d disabled=%d unique_names=%d duplicate_name_entries=%d",
				pluginID,
				stats.Total,
				stats.Enabled,
				stats.Disabled,
				stats.UniqueToolNames,
				stats.DuplicateNameEntries,
			)
		}

		if err := pluginsvc.Plugin.SyncMCPPluginToOpenViking(ctx, pluginID); err != nil {
			log.Printf("sync mcp plugin failed plugin_id=%s err=%v", pluginID, err)
			failed = append(failed, pluginID)
			continue
		}

		success++
		log.Printf("sync mcp plugin succeeded plugin_id=%s", pluginID)
	}

	log.Printf("migration completed total=%d success=%d skipped=%d failed=%d", total, success, skipped, len(failed))
	if len(failed) > 0 {
		return fmt.Errorf("migration failed for plugins: %s", strings.Join(failed, ","))
	}
	return nil
}

// loadConfigWithFallback 加载配置文件，并支持多层目录运行回退。
func loadConfigWithFallback(configPath string) (*config.Config, string, error) {
	configPath = strings.TrimSpace(configPath)
	if configPath == "" {
		configPath = "config.yaml"
	}

	candidates := make([]string, 0, 3)
	appendCandidate := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		for _, item := range candidates {
			if item == candidate {
				return
			}
		}
		candidates = append(candidates, candidate)
	}
	appendCandidate(configPath)
	if !filepath.IsAbs(configPath) {
		appendCandidate(filepath.Join("..", configPath))
		appendCandidate(filepath.Join("..", "..", configPath))
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err != nil {
			continue
		}
		cfg := config.LoadConfig(candidate)
		return cfg, candidate, nil
	}

	cwd, _ := os.Getwd()
	return nil, "", fmt.Errorf("config file not found, cwd=%s candidates=%s", cwd, strings.Join(candidates, ","))
}

// resolvePluginIDs 返回待迁移插件 ID 集合。
func resolvePluginIDs(pluginIDs []string) ([]string, error) {
	pluginIDs = normalizeUniqueList(pluginIDs)
	if len(pluginIDs) > 0 {
		return pluginIDs, nil
	}

	plugins, err := dao.Plugin.ListAll()
	if err != nil {
		return nil, fmt.Errorf("list plugins failed: %w", err)
	}
	resolved := make([]string, 0, len(plugins))
	for _, plugin := range plugins {
		if !isMCPRuntime(plugin.RuntimeType) {
			continue
		}
		resolved = append(resolved, plugin.PluginID)
	}
	return normalizeUniqueList(resolved), nil
}

// isMCPRuntime 判断 runtime_type 是否为 mcp。
func isMCPRuntime(runtimeType string) bool {
	return strings.EqualFold(strings.TrimSpace(runtimeType), pluginRuntimeMCP)
}

// splitCommaList 将逗号分隔参数转换为字符串切片。
func splitCommaList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			result = append(result, item)
		}
	}
	return result
}

// normalizeUniqueList 去重并移除空白值。
func normalizeUniqueList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

// collectPluginToolStats 汇总插件在 MySQL 中的工具数量信息。
func collectPluginToolStats(pluginID string) (pluginToolStats, error) {
	tools, err := dao.Plugin.ListToolsByPluginID(pluginID)
	if err != nil {
		return pluginToolStats{}, err
	}

	stats := pluginToolStats{Total: len(tools)}
	nameCounter := make(map[string]int, len(tools))
	for _, tool := range tools {
		if tool.Enabled {
			stats.Enabled++
		} else {
			stats.Disabled++
		}
		name := strings.TrimSpace(tool.ToolName)
		if name == "" {
			continue
		}
		nameCounter[name]++
	}
	stats.UniqueToolNames = len(nameCounter)
	for _, count := range nameCounter {
		if count > 1 {
			stats.DuplicateNameEntries += count - 1
		}
	}
	return stats, nil
}
