package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"openIntern/internal/config"
	"openIntern/internal/dao"
	"openIntern/internal/database"
	"openIntern/internal/models"

	"github.com/redis/go-redis/v9"
)

const (
	openVikingSyncQueueRedisKey       = "openintern:plugin:openviking:sync"
	defaultOpenVikingSyncDelay        = 2 * time.Second
	defaultOpenVikingSyncPollInterval = 3 * time.Second
	defaultOpenVikingSyncTimeout      = 30 * time.Second
	defaultOpenVikingSyncRetryDelay   = 30 * time.Second
	defaultOpenVikingSyncWorkerBatch  = 20

	openVikingSyncTaskReconcilePrefix = "reconcile:"
	openVikingSyncTaskCleanupPrefix   = "cleanup:"

	maxIndexedCodeLength = 6000
)

var (
	openVikingSyncDelay        = defaultOpenVikingSyncDelay
	openVikingSyncPollInterval = defaultOpenVikingSyncPollInterval
	openVikingSyncTimeout      = defaultOpenVikingSyncTimeout
	openVikingSyncRetryDelay   = defaultOpenVikingSyncRetryDelay

	openVikingSyncStartOnce sync.Once
)

// initPluginOpenVikingSync 初始化 MySQL -> OpenViking 同步流水线。
func initPluginOpenVikingSync(cfg config.PluginConfig) {
	openVikingSyncDelay = durationFromSeconds(cfg.OpenVikingSyncDelaySeconds, defaultOpenVikingSyncDelay)
	openVikingSyncPollInterval = durationFromSeconds(cfg.OpenVikingSyncPollSeconds, defaultOpenVikingSyncPollInterval)
	openVikingSyncTimeout = durationFromSeconds(cfg.OpenVikingSyncTimeoutSeconds, defaultOpenVikingSyncTimeout)
	openVikingSyncRetryDelay = durationFromSeconds(cfg.OpenVikingSyncRetrySeconds, defaultOpenVikingSyncRetryDelay)

	openVikingSyncStartOnce.Do(func() {
		go Plugin.runOpenVikingSyncQueueWorker()
	})
}

func buildOpenVikingSyncTask(action string, userID string, pluginID string) string {
	member := encodePluginQueueMember(userID, pluginID)
	if member == "" {
		return ""
	}
	return action + member
}

// queueOpenVikingPluginReconcile 入队插件对账任务，依据 MySQL 当前数据重建 OpenViking 工具索引。
func (s *PluginService) queueOpenVikingPluginReconcile(userID string, pluginID string, delay time.Duration) error {
	taskKey := buildOpenVikingSyncTask(openVikingSyncTaskReconcilePrefix, userID, pluginID)
	if taskKey == "" || !dao.Plugin.ToolStoreConfigured() {
		return nil
	}
	return s.queueOpenVikingSyncTask(taskKey, delay)
}

// queueOpenVikingPluginCleanup 入队插件清理任务，删除 OpenViking 中该插件目录。
func (s *PluginService) queueOpenVikingPluginCleanup(userID string, pluginID string, delay time.Duration) error {
	taskKey := buildOpenVikingSyncTask(openVikingSyncTaskCleanupPrefix, userID, pluginID)
	if taskKey == "" || !dao.Plugin.ToolStoreConfigured() {
		return nil
	}
	return s.queueOpenVikingSyncTask(taskKey, delay)
}

// clearOpenVikingPluginTasks 清理队列中的插件任务。
func (s *PluginService) clearOpenVikingPluginTasks(userID string, pluginID string) {
	reconcileTask := buildOpenVikingSyncTask(openVikingSyncTaskReconcilePrefix, userID, pluginID)
	cleanupTask := buildOpenVikingSyncTask(openVikingSyncTaskCleanupPrefix, userID, pluginID)
	if reconcileTask == "" || cleanupTask == "" {
		return
	}
	redisClient := database.GetRedis()
	if redisClient == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := redisClient.ZRem(
		ctx,
		openVikingSyncQueueRedisKey,
		reconcileTask,
		cleanupTask,
	).Err(); err != nil {
		log.Printf("clear openviking sync queue failed user_id=%s plugin_id=%s err=%v", userID, pluginID, err)
	}
}

// queueOpenVikingSyncTask 将指定任务写入 Redis 延迟队列。
func (s *PluginService) queueOpenVikingSyncTask(taskKey string, delay time.Duration) error {
	taskKey = strings.TrimSpace(taskKey)
	if taskKey == "" || !dao.Plugin.ToolStoreConfigured() {
		return nil
	}
	if delay < 0 {
		delay = 0
	}

	redisClient := database.GetRedis()
	if redisClient == nil {
		return errors.New("openviking sync queue requires redis")
	}

	runAt := time.Now().Add(delay).UnixMilli()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return redisClient.ZAdd(ctx, openVikingSyncQueueRedisKey, redis.Z{
		Score:  float64(runAt),
		Member: taskKey,
	}).Err()
}

// runOpenVikingSyncQueueWorker 周期消费同步队列。
func (s *PluginService) runOpenVikingSyncQueueWorker() {
	if openVikingSyncPollInterval <= 0 {
		return
	}
	ticker := time.NewTicker(openVikingSyncPollInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.processQueuedOpenVikingSyncTasks(context.Background(), defaultOpenVikingSyncWorkerBatch); err != nil {
			log.Printf("process queued openviking sync tasks failed err=%v", err)
		}
	}
}

// processQueuedOpenVikingSyncTasks 批量拉取到期任务并执行。
func (s *PluginService) processQueuedOpenVikingSyncTasks(ctx context.Context, limit int64) error {
	redisClient := database.GetRedis()
	if redisClient == nil {
		return errors.New("openviking sync queue requires redis")
	}
	if limit <= 0 {
		limit = defaultOpenVikingSyncWorkerBatch
	}

	readCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	items, err := redisClient.ZRangeByScore(readCtx, openVikingSyncQueueRedisKey, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   strconv.FormatInt(time.Now().UnixMilli(), 10),
		Count: limit,
	}).Result()
	if err != nil {
		return err
	}

	for _, taskKey := range items {
		removeCtx, removeCancel := context.WithTimeout(context.Background(), 2*time.Second)
		removed, removeErr := redisClient.ZRem(removeCtx, openVikingSyncQueueRedisKey, taskKey).Result()
		removeCancel()
		if removeErr != nil {
			log.Printf("remove openviking sync task failed task=%s err=%v", taskKey, removeErr)
			continue
		}
		if removed == 0 {
			continue
		}
		if err := s.processOpenVikingSyncTask(context.Background(), taskKey); err != nil {
			log.Printf("run openviking sync task failed task=%s err=%v", taskKey, err)
			if openVikingSyncRetryDelay > 0 {
				if queueErr := s.queueOpenVikingSyncTask(taskKey, openVikingSyncRetryDelay); queueErr != nil {
					log.Printf("requeue openviking sync task failed task=%s err=%v", taskKey, queueErr)
				}
			}
		}
	}
	return nil
}

// processOpenVikingSyncTask 解析并执行单个同步任务。
func (s *PluginService) processOpenVikingSyncTask(ctx context.Context, taskKey string) error {
	action, userID, pluginID, ok := parseOpenVikingSyncTask(taskKey)
	if !ok {
		return fmt.Errorf("invalid openviking sync task: %s", taskKey)
	}

	runCtx := ctx
	cancel := func() {}
	if runCtx == nil {
		runCtx = context.Background()
	}
	if _, hasDeadline := runCtx.Deadline(); !hasDeadline && openVikingSyncTimeout > 0 {
		runCtx, cancel = context.WithTimeout(runCtx, openVikingSyncTimeout)
	}
	defer cancel()

	switch action {
	case openVikingSyncTaskReconcilePrefix:
		return s.syncPluginToolsToOpenVikingNow(runCtx, userID, pluginID)
	case openVikingSyncTaskCleanupPrefix:
		return dao.Plugin.DeleteToolStorePluginURI(runCtx, userID, pluginID)
	default:
		return fmt.Errorf("unsupported openviking sync action: %s", action)
	}
}

// parseOpenVikingSyncTask 解析任务前缀与插件标识。
func parseOpenVikingSyncTask(taskKey string) (string, string, string, bool) {
	taskKey = strings.TrimSpace(taskKey)
	if strings.HasPrefix(taskKey, openVikingSyncTaskReconcilePrefix) {
		return decodeOpenVikingSyncTask(taskKey, openVikingSyncTaskReconcilePrefix)
	}
	if strings.HasPrefix(taskKey, openVikingSyncTaskCleanupPrefix) {
		return decodeOpenVikingSyncTask(taskKey, openVikingSyncTaskCleanupPrefix)
	}
	return "", "", "", false
}

func decodeOpenVikingSyncTask(taskKey string, prefix string) (string, string, string, bool) {
	userID, pluginID, ok := decodePluginQueueMember(strings.TrimPrefix(strings.TrimSpace(taskKey), prefix))
	if !ok {
		return "", "", "", false
	}
	return prefix, userID, pluginID, true
}

// SyncPluginToolsToOpenViking 立即执行单插件同步，供运维脚本与手动回填使用。
func (s *PluginService) SyncPluginToolsToOpenViking(ctx context.Context, userID string, pluginID string) error {
	pluginID = strings.TrimSpace(pluginID)
	userID = strings.TrimSpace(userID)
	if userID == "" || pluginID == "" {
		return errors.New("user_id and plugin_id are required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return s.syncPluginToolsToOpenVikingNow(ctx, userID, pluginID)
}

// syncPluginToolsToOpenVikingNow 使用 MySQL 工具快照对账 OpenViking 资源目录。
func (s *PluginService) syncPluginToolsToOpenVikingNow(ctx context.Context, userID string, pluginID string) error {
	if !dao.Plugin.ToolStoreConfigured() {
		return nil
	}

	plugin, err := s.getPluginRecord(userID, pluginID)
	if err != nil {
		if errors.Is(err, dao.ErrPluginNotFound) {
			return dao.Plugin.DeleteToolStorePluginURI(ctx, userID, pluginID)
		}
		return err
	}

	tools, err := dao.Plugin.ListToolsByUserIDAndPluginID(userID, pluginID)
	if err != nil {
		return err
	}

	expected := buildExpectedToolStoreDocuments(plugin, tools)
	return dao.Plugin.ReplaceToolStoreResourcesByPlugin(ctx, plugin.UserID, plugin.PluginID, expected)
}

// buildExpectedToolStoreDocuments 构建插件期望写入的工具文档集合。
func buildExpectedToolStoreDocuments(plugin *models.Plugin, tools []models.Tool) map[string]string {
	documents := make(map[string]string)
	if plugin == nil || !shouldIndexPlugin(plugin) {
		return documents
	}
	for _, tool := range tools {
		if !shouldIndexTool(plugin, tool) {
			continue
		}
		documents[tool.ToolID] = buildOpenVikingToolDocument(plugin, tool)
	}
	return documents
}

// shouldIndexPlugin 判断插件是否参与工具检索索引。
func shouldIndexPlugin(plugin *models.Plugin) bool {
	return plugin != nil && strings.EqualFold(strings.TrimSpace(plugin.Status), pluginStatusEnabled)
}

// shouldIndexTool 判断工具是否参与工具检索索引。
func shouldIndexTool(plugin *models.Plugin, tool models.Tool) bool {
	if !shouldIndexPlugin(plugin) {
		return false
	}
	if !tool.Enabled {
		return false
	}
	return strings.TrimSpace(tool.ToolID) != ""
}

// buildOpenVikingToolDocument 构造写入 OpenViking 的工具检索文档。
func buildOpenVikingToolDocument(plugin *models.Plugin, tool models.Tool) string {
	var builder strings.Builder
	builder.WriteString("# Tool\n")
	appendToolDocLine(&builder, "tool_id", tool.ToolID)
	appendToolDocLine(&builder, "tool_name", tool.ToolName)
	appendToolDocLine(&builder, "description", tool.Description)
	appendToolDocLine(&builder, "runtime_type", plugin.RuntimeType)
	appendToolDocLine(&builder, "plugin_id", plugin.PluginID)
	appendToolDocLine(&builder, "plugin_name", plugin.Name)
	appendToolDocLine(&builder, "plugin_description", plugin.Description)

	appendToolDocLine(&builder, "tool_response_mode", tool.ToolResponseMode)
	appendToolDocLine(&builder, "input_schema_json", compactJSONString(tool.InputSchemaJSON))
	appendToolDocLine(&builder, "output_schema_json", compactJSONString(tool.OutputSchemaJSON))

	switch strings.ToLower(strings.TrimSpace(plugin.RuntimeType)) {
	case pluginRuntimeAPI:
		appendToolDocLine(&builder, "api_request_type", tool.APIRequestType)
		appendToolDocLine(&builder, "request_url", tool.RequestURL)
		appendToolDocLine(&builder, "query_fields_json", compactJSONString(tool.QueryFieldsJSON))
		appendToolDocLine(&builder, "header_fields_json", compactJSONString(tool.HeaderFieldsJSON))
		appendToolDocLine(&builder, "body_fields_json", compactJSONString(tool.BodyFieldsJSON))
	case pluginRuntimeCode:
		appendToolDocLine(&builder, "code_language", tool.CodeLanguage)
		appendToolDocLine(&builder, "code", truncateText(strings.TrimSpace(tool.Code), maxIndexedCodeLength))
	case pluginRuntimeMCP:
		appendToolDocLine(&builder, "mcp_url", plugin.MCPURL)
		appendToolDocLine(&builder, "mcp_protocol", plugin.MCPProtocol)
	}
	return builder.String()
}

// appendToolDocLine 以 `key: value` 形式追加文档行。
func appendToolDocLine(builder *strings.Builder, key string, value string) {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if builder == nil || key == "" || value == "" {
		return
	}
	builder.WriteString("- ")
	builder.WriteString(key)
	builder.WriteString(": ")
	builder.WriteString(value)
	builder.WriteString("\n")
}

// compactJSONString 压缩 JSON 字符串，失败时返回原始去空白文本。
func compactJSONString(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, []byte(raw)); err != nil {
		return raw
	}
	return compact.String()
}

// truncateText 按字符数截断文本。
func truncateText(value string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxLen {
		return value
	}
	return string(runes[:maxLen])
}
