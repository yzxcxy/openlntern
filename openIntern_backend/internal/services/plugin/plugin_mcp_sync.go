package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"openIntern/internal/config"
	"openIntern/internal/dao"
	"openIntern/internal/database"
	"openIntern/internal/models"
	"openIntern/internal/util"

	mcpTool "github.com/cloudwego/eino-ext/components/tool/mcp"
	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/redis/go-redis/v9"
)

const (
	mcpSyncQueueRedisKey        = "openintern:plugin:mcp:sync"
	defaultMCPSyncDelay         = 3 * time.Second
	defaultMCPSyncPollInterval  = 3 * time.Second
	defaultMCPSyncRefreshWindow = 10 * time.Minute
	defaultMCPSyncTimeout       = 30 * time.Second
	defaultMCPSyncRetryDelay    = time.Minute
	defaultMCPSyncWorkerBatch   = 5
)

var (
	errMCPPluginSyncBusy = errors.New("mcp plugin sync is already running")

	mcpSyncDelay         = defaultMCPSyncDelay
	mcpSyncPollInterval  = defaultMCPSyncPollInterval
	mcpSyncRefreshWindow = defaultMCPSyncRefreshWindow
	mcpSyncTimeout       = defaultMCPSyncTimeout
	mcpSyncRetryDelay    = defaultMCPSyncRetryDelay

	mcpSyncStartOnce sync.Once
)

var enqueueMCPPluginSyncScript = `
local key = KEYS[1]
local member = ARGV[1]
local score = tonumber(ARGV[2])
local existing = redis.call("ZSCORE", key, member)
if not existing then
	redis.call("ZADD", key, score, member)
	return 1
end
if score < tonumber(existing) then
	redis.call("ZADD", key, score, member)
	return 1
end
return 0
`

var seedMCPPluginSyncScript = `
local key = KEYS[1]
local member = ARGV[1]
local score = tonumber(ARGV[2])
local existing = redis.call("ZSCORE", key, member)
if existing then
	return 0
end
redis.call("ZADD", key, score, member)
return 1
`

var finalizeMCPPluginSyncScript = `
local key = KEYS[1]
local member = ARGV[1]
local processing = tonumber(ARGV[2])
local next_score = tonumber(ARGV[3])
local current = redis.call("ZSCORE", key, member)
if not current then
	return 0
end
current = tonumber(current)
if current == processing then
	redis.call("ZADD", key, next_score, member)
	return 1
end
if current > next_score then
	redis.call("ZADD", key, next_score, member)
	return 1
end
return 0
`

func encodePluginQueueMember(userID string, pluginID string) string {
	userID = strings.TrimSpace(userID)
	pluginID = strings.TrimSpace(pluginID)
	if userID == "" || pluginID == "" {
		return ""
	}
	return userID + ":" + pluginID
}

func decodePluginQueueMember(value string) (string, string, bool) {
	value = strings.TrimSpace(value)
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	userID := strings.TrimSpace(parts[0])
	pluginID := strings.TrimSpace(parts[1])
	return userID, pluginID, userID != "" && pluginID != ""
}

func initPluginMCPSync(cfg config.PluginConfig) {
	mcpSyncDelay = durationFromSeconds(cfg.MCPSyncDelaySeconds, defaultMCPSyncDelay)
	mcpSyncPollInterval = durationFromSeconds(cfg.MCPSyncPollSeconds, defaultMCPSyncPollInterval)
	mcpSyncRefreshWindow = durationFromSeconds(cfg.MCPSyncIntervalSeconds, defaultMCPSyncRefreshWindow)
	mcpSyncTimeout = durationFromSeconds(cfg.MCPSyncTimeoutSeconds, defaultMCPSyncTimeout)
	mcpSyncRetryDelay = durationFromSeconds(cfg.MCPSyncRetrySeconds, defaultMCPSyncRetryDelay)

	mcpSyncStartOnce.Do(func() {
		go func() {
			if err := Plugin.scheduleAllEnabledMCPSyncs(); err != nil {
				log.Printf("initial full mcp sync scheduling failed err=%v", err)
			}
		}()
		go Plugin.runMCPSyncQueueWorker()
	})
}

func durationFromSeconds(seconds int, fallback time.Duration) time.Duration {
	if seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

func (s *PluginService) queueMCPPluginSync(userID string, pluginID string, delay time.Duration) error {
	member := encodePluginQueueMember(userID, pluginID)
	if member == "" {
		return nil
	}
	if delay < 0 {
		delay = 0
	}

	redisClient := database.GetRedis()
	if redisClient == nil {
		return errors.New("mcp sync queue requires redis")
	}

	return runMCPPluginSyncQueueScript(redisClient, enqueueMCPPluginSyncScript, member, time.Now().Add(delay).UnixMilli())
}

func (s *PluginService) seedMCPPluginSync(userID string, pluginID string, delay time.Duration) error {
	member := encodePluginQueueMember(userID, pluginID)
	if member == "" {
		return nil
	}
	if delay < 0 {
		delay = 0
	}

	redisClient := database.GetRedis()
	if redisClient == nil {
		return errors.New("mcp sync queue requires redis")
	}

	return runMCPPluginSyncQueueScript(redisClient, seedMCPPluginSyncScript, member, time.Now().Add(delay).UnixMilli())
}

func (s *PluginService) clearMCPPluginSync(userID string, pluginID string) {
	member := encodePluginQueueMember(userID, pluginID)
	if member == "" {
		return
	}
	redisClient := database.GetRedis()
	if redisClient == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := redisClient.ZRem(ctx, mcpSyncQueueRedisKey, member).Err(); err != nil {
		log.Printf("clear mcp sync queue failed user_id=%s plugin_id=%s err=%v", userID, pluginID, err)
	}
}

func runMCPPluginSyncQueueScript(redisClient *redis.Client, script string, member string, runAt int64) error {
	if redisClient == nil {
		return errors.New("mcp sync queue requires redis")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return redisClient.Eval(
		ctx,
		script,
		[]string{mcpSyncQueueRedisKey},
		member,
		strconv.FormatInt(runAt, 10),
	).Err()
}

func (s *PluginService) setMCPPluginSyncRunAt(userID string, pluginID string, runAt int64) error {
	member := encodePluginQueueMember(userID, pluginID)
	if member == "" {
		return nil
	}

	redisClient := database.GetRedis()
	if redisClient == nil {
		return errors.New("mcp sync queue requires redis")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return redisClient.ZAdd(ctx, mcpSyncQueueRedisKey, redis.Z{
		Score:  float64(runAt),
		Member: member,
	}).Err()
}

func (s *PluginService) markMCPPluginSyncInProgress(userID string, pluginID string) (int64, error) {
	runAt := time.Now().Add(currentMCPSyncLockTTL()).UnixMilli()
	return runAt, s.setMCPPluginSyncRunAt(userID, pluginID, runAt)
}

func (s *PluginService) finalizeMCPPluginSync(userID string, pluginID string, processingRunAt int64, failed bool) error {
	delay, ok := s.nextMCPSyncDelay(userID, pluginID, failed)
	if !ok {
		s.clearMCPPluginSync(userID, pluginID)
		return nil
	}

	redisClient := database.GetRedis()
	if redisClient == nil {
		return errors.New("mcp sync queue requires redis")
	}

	nextRunAt := time.Now().Add(delay).UnixMilli()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	member := encodePluginQueueMember(userID, pluginID)
	return redisClient.Eval(
		ctx,
		finalizeMCPPluginSyncScript,
		[]string{mcpSyncQueueRedisKey},
		member,
		strconv.FormatInt(processingRunAt, 10),
		strconv.FormatInt(nextRunAt, 10),
	).Err()
}

func (s *PluginService) runMCPSyncQueueWorker() {
	if mcpSyncPollInterval <= 0 {
		return
	}
	ticker := time.NewTicker(mcpSyncPollInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.processQueuedMCPSyncTasks(context.Background(), defaultMCPSyncWorkerBatch); err != nil {
			log.Printf("process queued mcp sync tasks failed err=%v", err)
		}
	}
}

func (s *PluginService) processQueuedMCPSyncTasks(ctx context.Context, limit int64) error {
	redisClient := database.GetRedis()
	if redisClient == nil {
		return errors.New("mcp sync queue requires redis")
	}
	if limit <= 0 {
		limit = defaultMCPSyncWorkerBatch
	}

	readCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	ids, err := redisClient.ZRangeByScore(readCtx, mcpSyncQueueRedisKey, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   strconv.FormatInt(time.Now().UnixMilli(), 10),
		Count: limit,
	}).Result()
	if err != nil {
		return err
	}

	for _, member := range ids {
		s.triggerScheduledMCPSync(member)
	}

	return nil
}

// scheduleAllEnabledMCPSyncs 在进程启动时将全部启用中的 MCP 插件放入延时队列，
// 确保队列具备完整初始集，后续由 worker 在成功/失败后自行续期。
func (s *PluginService) scheduleAllEnabledMCPSyncs() error {
	plugins, err := dao.Plugin.ListByRuntimeStatusAll(pluginRuntimeMCP, pluginStatusEnabled)
	if err != nil {
		return err
	}

	for _, plugin := range plugins {
		if err := s.seedMCPPluginSync(plugin.UserID, plugin.PluginID, 0); err != nil {
			log.Printf("enqueue initial mcp sync failed user_id=%s plugin_id=%s err=%v", plugin.UserID, plugin.PluginID, err)
		}
	}

	return nil
}

func (s *PluginService) triggerScheduledMCPSync(member string) {
	userID, pluginID, ok := decodePluginQueueMember(member)
	if !ok {
		return
	}
	lock, acquired, err := tryAcquireMCPPluginSyncLock(userID, pluginID)
	if err != nil {
		log.Printf("acquire mcp sync lock failed user_id=%s plugin_id=%s err=%v", userID, pluginID, err)
		return
	}
	if !acquired {
		return
	}
	defer func() {
		if releaseErr := lock.Release(); releaseErr != nil {
			log.Printf("release mcp sync lock failed user_id=%s plugin_id=%s err=%v", userID, pluginID, releaseErr)
		}
	}()

	plugin, err := s.getPluginRecord(userID, pluginID)
	if err != nil {
		if errors.Is(err, dao.ErrPluginNotFound) {
			s.clearMCPPluginSync(userID, pluginID)
			return
		}
		log.Printf("load mcp plugin failed user_id=%s plugin_id=%s err=%v", userID, pluginID, err)
		return
	}
	if plugin.RuntimeType != pluginRuntimeMCP || plugin.Status != pluginStatusEnabled {
		s.clearMCPPluginSync(userID, pluginID)
		return
	}

	processingRunAt, err := s.markMCPPluginSyncInProgress(userID, pluginID)
	if err != nil {
		log.Printf("mark mcp sync in progress failed user_id=%s plugin_id=%s err=%v", userID, pluginID, err)
		return
	}

	syncCtx, cancel := buildMCPSyncContext(context.Background())
	defer cancel()

	syncErr := s.syncMCPPluginRecord(syncCtx, plugin)
	if syncErr != nil {
		log.Printf("scheduled mcp sync failed user_id=%s plugin_id=%s err=%v", userID, pluginID, syncErr)
	}
	if finalizeErr := s.finalizeMCPPluginSync(userID, pluginID, processingRunAt, syncErr != nil); finalizeErr != nil {
		log.Printf("finalize mcp sync schedule failed user_id=%s plugin_id=%s err=%v", userID, pluginID, finalizeErr)
	}
}

func (s *PluginService) nextMCPSyncDelay(userID string, pluginID string, failed bool) (time.Duration, bool) {
	plugin, err := s.getPluginRecord(userID, pluginID)
	if err != nil {
		return 0, false
	}
	if plugin.RuntimeType != pluginRuntimeMCP || plugin.Status != pluginStatusEnabled {
		return 0, false
	}

	if failed {
		delay := mcpSyncRefreshWindow
		if mcpSyncRetryDelay > 0 {
			delay += mcpSyncRetryDelay
		}
		return delay, true
	}

	if mcpSyncRefreshWindow <= 0 {
		return 0, false
	}
	return mcpSyncRefreshWindow, true
}

func (s *PluginService) syncMCPPluginNow(ctx context.Context, userID string, pluginID string, allowDisabled bool) error {
	lock, acquired, err := tryAcquireMCPPluginSyncLock(userID, pluginID)
	if err != nil {
		return err
	}
	if !acquired {
		return errMCPPluginSyncBusy
	}
	defer func() {
		if releaseErr := lock.Release(); releaseErr != nil {
			log.Printf("release mcp sync lock failed user_id=%s plugin_id=%s err=%v", userID, pluginID, releaseErr)
		}
	}()

	return s.syncMCPPluginNowLocked(ctx, userID, pluginID, allowDisabled)
}

func buildMCPSyncContext(ctx context.Context) (context.Context, func()) {
	syncCtx := ctx
	if syncCtx == nil {
		syncCtx = context.Background()
	}
	if _, hasDeadline := syncCtx.Deadline(); hasDeadline || mcpSyncTimeout <= 0 {
		return syncCtx, func() {}
	}

	return context.WithTimeout(syncCtx, mcpSyncTimeout)
}

func (s *PluginService) syncMCPPluginNowLocked(ctx context.Context, userID string, pluginID string, allowDisabled bool) error {
	plugin, err := s.getPluginRecord(userID, pluginID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}
	if plugin.RuntimeType != pluginRuntimeMCP {
		return nil
	}
	if !allowDisabled && plugin.Status != pluginStatusEnabled {
		return nil
	}

	syncCtx, cancel := buildMCPSyncContext(ctx)
	defer cancel()

	return s.syncMCPPluginRecord(syncCtx, plugin)
}

// SyncMCPPluginTools 立即拉取 MCP 工具并同步到 MySQL，供运维脚本复用。
func (s *PluginService) SyncMCPPluginTools(ctx context.Context, userID string, pluginID string) error {
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return errors.New("plugin_id is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	plugin, err := s.getPluginRecord(userID, pluginID)
	if err != nil {
		return err
	}
	if plugin.RuntimeType != pluginRuntimeMCP {
		return errors.New("only mcp plugins support sync")
	}

	lock, acquired, err := tryAcquireMCPPluginSyncLock(userID, pluginID)
	if err != nil {
		return err
	}
	if !acquired {
		return errMCPPluginSyncBusy
	}
	defer func() {
		if releaseErr := lock.Release(); releaseErr != nil {
			log.Printf("release mcp sync lock failed user_id=%s plugin_id=%s err=%v", userID, pluginID, releaseErr)
		}
	}()

	syncCtx, cancel := buildMCPSyncContext(ctx)
	defer cancel()

	return s.syncMCPPluginRecord(syncCtx, plugin)
}

// syncMCPPluginRecord 同步 MCP 插件工具快照到 MySQL。
func (s *PluginService) syncMCPPluginRecord(ctx context.Context, plugin *models.Plugin) error {
	if plugin == nil {
		return fmt.Errorf("plugin not found")
	}

	remoteTools, err := fetchMCPToolDefinitions(ctx, plugin.MCPURL, plugin.MCPProtocol)
	if err != nil {
		return err
	}

	existingTools, err := dao.Plugin.ListToolsByUserIDAndPluginID(plugin.UserID, plugin.PluginID)
	if err != nil {
		return err
	}

	syncedTools, err := buildSyncedMCPToolRecords(plugin.UserID, plugin.PluginID, remoteTools, existingTools)
	if err != nil {
		return err
	}
	syncedAt := time.Now()

	if !hasMCPToolRecordChanges(existingTools, syncedTools) {
		if err := dao.Plugin.UpdateLastSyncAt(plugin.UserID, plugin.PluginID, syncedAt); err != nil {
			return err
		}
		return nil
	}

	if err := dao.Plugin.ReplaceToolsAndUpdateSyncTime(plugin.UserID, plugin.PluginID, syncedTools, syncedAt); err != nil {
		return err
	}
	return nil
}

type mcpToolComparable struct {
	ToolID           string
	PluginID         string
	ToolName         string
	Description      string
	InputSchemaJSON  string
	OutputSchemaJSON string
	ToolResponseMode string
	Enabled          bool
	TimeoutMS        int
}

// hasMCPToolRecordChanges 比较 MCP 工具快照是否发生变化。
func hasMCPToolRecordChanges(existingTools []models.Tool, syncedTools []models.Tool) bool {
	if len(existingTools) != len(syncedTools) {
		return true
	}
	existingMap := make(map[string]mcpToolComparable, len(existingTools))
	for _, item := range existingTools {
		existingMap[item.ToolID] = toMCPToolComparable(item)
	}
	for _, item := range syncedTools {
		existing, ok := existingMap[item.ToolID]
		if !ok {
			return true
		}
		if existing != toMCPToolComparable(item) {
			return true
		}
	}
	return false
}

// toMCPToolComparable 仅提取 MCP 同步判定所需字段。
func toMCPToolComparable(item models.Tool) mcpToolComparable {
	return mcpToolComparable{
		ToolID:           strings.TrimSpace(item.ToolID),
		PluginID:         strings.TrimSpace(item.PluginID),
		ToolName:         strings.TrimSpace(item.ToolName),
		Description:      strings.TrimSpace(item.Description),
		InputSchemaJSON:  strings.TrimSpace(item.InputSchemaJSON),
		OutputSchemaJSON: strings.TrimSpace(item.OutputSchemaJSON),
		ToolResponseMode: strings.TrimSpace(item.ToolResponseMode),
		Enabled:          item.Enabled,
		TimeoutMS:        item.TimeoutMS,
	}
}

func fetchMCPToolDefinitions(ctx context.Context, baseURL string, protocol string) ([]mcp.Tool, error) {
	cli, err := openMCPClient(ctx, baseURL, protocol)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cli.Close()
	}()

	result, err := cli.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list mcp tools failed: %w", err)
	}
	return result.Tools, nil
}

func openMCPClient(ctx context.Context, baseURL string, protocol string) (*client.Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("mcp_url is required for mcp plugins")
	}

	normalizedProtocol := normalizeMCPProtocol(protocol)
	if normalizedProtocol == "" {
		return nil, fmt.Errorf("mcp_protocol must be sse or streamableHttp")
	}

	protocols := []string{normalizedProtocol}
	switch normalizedProtocol {
	case mcpProtocolSSE:
		protocols = append(protocols, mcpProtocolStreamableHTTP)
	case mcpProtocolStreamableHTTP:
		protocols = append(protocols, mcpProtocolSSE)
	}

	var errs []error
	for _, candidate := range protocols {
		cli, err := openMCPClientWithProtocol(ctx, baseURL, candidate)
		if err == nil {
			if candidate != normalizedProtocol {
				log.Printf("mcp protocol fallback success base_url=%s configured=%s actual=%s", baseURL, normalizedProtocol, candidate)
			}
			return cli, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", candidate, err))
		if !shouldTryAlternateMCPProtocol(normalizedProtocol, candidate, err) {
			break
		}
	}

	return nil, errors.Join(errs...)
}

func openMCPClientWithProtocol(ctx context.Context, baseURL string, protocol string) (*client.Client, error) {
	var (
		cli *client.Client
		err error
	)

	switch protocol {
	case mcpProtocolSSE:
		cli, err = client.NewSSEMCPClient(baseURL)
	case mcpProtocolStreamableHTTP:
		cli, err = client.NewStreamableHttpClient(baseURL)
	default:
		return nil, fmt.Errorf("mcp_protocol must be sse or streamableHttp")
	}
	if err != nil {
		return nil, err
	}

	if err := cli.Start(ctx); err != nil {
		_ = cli.Close()
		return nil, err
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "openintern",
		Version: "1.0.0",
	}
	if _, err := cli.Initialize(ctx, initRequest); err != nil {
		_ = cli.Close()
		return nil, err
	}

	return cli, nil
}

func shouldTryAlternateMCPProtocol(configured string, attempted string, err error) bool {
	if err == nil || configured != attempted {
		return false
	}

	message := strings.ToLower(err.Error())
	switch attempted {
	case mcpProtocolSSE:
		return strings.Contains(message, "timeout waiting for endpoint") ||
			strings.Contains(message, "endpoint not received")
	case mcpProtocolStreamableHTTP:
		return strings.Contains(message, "method not allowed") ||
			strings.Contains(message, "unexpected status code: 404") ||
			strings.Contains(message, "unexpected status code: 405")
	default:
		return false
	}
}

func buildSyncedMCPToolRecords(userID string, pluginID string, remoteTools []mcp.Tool, existingTools []models.Tool) ([]models.Tool, error) {
	byName := make(map[string]models.Tool, len(existingTools))
	for _, tool := range existingTools {
		byName[tool.ToolName] = tool
	}

	defaultOutputSchema, err := normalizeOutputSchema("")
	if err != nil {
		return nil, err
	}

	synced := make([]models.Tool, 0, len(remoteTools))
	for _, remoteTool := range remoteTools {
		toolName := strings.TrimSpace(remoteTool.Name)
		if toolName == "" {
			continue
		}

		inputSchemaJSON, err := marshalMCPToolSchema(remoteTool.InputSchema)
		if err != nil {
			return nil, fmt.Errorf("marshal input schema for %s: %w", toolName, err)
		}

		outputSchemaJSON := defaultOutputSchema
		if hasMCPOutputSchema(remoteTool) {
			outputSchemaJSON, err = marshalMCPToolSchema(remoteTool.OutputSchema)
			if err != nil {
				return nil, fmt.Errorf("marshal output schema for %s: %w", toolName, err)
			}
		}

		existing := byName[toolName]
		toolID := strings.TrimSpace(existing.ToolID)
		if toolID == "" {
			toolID = uuid.NewString()
		}

		toolResponseMode := strings.TrimSpace(existing.ToolResponseMode)
		if toolResponseMode == "" {
			toolResponseMode = toolResponseNonStreaming
		}

		timeoutMS := existing.TimeoutMS
		if timeoutMS <= 0 {
			timeoutMS = defaultPluginTimeoutMS
		}

		synced = append(synced, models.Tool{
			UserID:           userID,
			ToolID:           toolID,
			PluginID:         pluginID,
			ToolName:         toolName,
			Description:      strings.TrimSpace(remoteTool.Description),
			InputSchemaJSON:  inputSchemaJSON,
			OutputSchemaJSON: outputSchemaJSON,
			ToolResponseMode: toolResponseMode,
			Enabled:          existing.ToolID == "" || existing.Enabled,
			TimeoutMS:        timeoutMS,
		})
	}

	sort.SliceStable(synced, func(i, j int) bool {
		return synced[i].ToolName < synced[j].ToolName
	})
	return synced, nil
}

func marshalMCPToolSchema(schemaValue any) (string, error) {
	raw, err := json.Marshal(schemaValue)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func hasMCPOutputSchema(tool mcp.Tool) bool {
	return tool.OutputSchema.Type != "" ||
		len(tool.OutputSchema.Properties) > 0 ||
		len(tool.OutputSchema.Required) > 0 ||
		len(tool.OutputSchema.Defs) > 0 ||
		tool.OutputSchema.AdditionalProperties != nil
}

func (s *PluginService) BuildRuntimeMCPTools(ctx context.Context, toolIDs []string) ([]einoTool.BaseTool, func(), error) {
	toolIDs = util.NormalizeUniqueStringList(toolIDs)
	if len(toolIDs) == 0 {
		return nil, nil, nil
	}
	return s.buildRuntimeMCPTools(ctx, toolIDs)
}

// BuildAllRuntimeMCPTools 构建全部启用态 mcp 插件工具，供动态工具池预装使用。
func (s *PluginService) BuildAllRuntimeMCPTools(ctx context.Context) ([]einoTool.BaseTool, func(), error) {
	return s.buildRuntimeMCPTools(ctx, nil)
}

func (s *PluginService) buildRuntimeMCPTools(ctx context.Context, toolIDs []string) ([]einoTool.BaseTool, func(), error) {
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, nil, nil
	}
	toolRows, err := dao.Plugin.ListRuntimeTools(userID, pluginRuntimeMCP, pluginStatusEnabled, toolIDs)
	if err != nil {
		return nil, nil, err
	}
	if len(toolRows) == 0 {
		return nil, nil, nil
	}

	pluginIDSet := make(map[string]struct{}, len(toolRows))
	toolNamesByPlugin := make(map[string][]string, len(toolRows))
	for _, tool := range toolRows {
		pluginIDSet[tool.PluginID] = struct{}{}
		toolNamesByPlugin[tool.PluginID] = append(toolNamesByPlugin[tool.PluginID], tool.ToolName)
	}

	pluginIDList := make([]string, 0, len(pluginIDSet))
	for pluginID := range pluginIDSet {
		pluginIDList = append(pluginIDList, pluginID)
	}
	sort.Strings(pluginIDList)

	plugins, err := dao.Plugin.ListByIDsAndRuntimeStatus(userID, pluginIDList, pluginRuntimeMCP, pluginStatusEnabled)
	if err != nil {
		return nil, nil, err
	}

	pluginByID := make(map[string]models.Plugin, len(plugins))
	for _, plugin := range plugins {
		pluginByID[plugin.PluginID] = plugin
	}

	var (
		runtimeTools []einoTool.BaseTool
		closers      []*client.Client
	)

	for _, pluginID := range pluginIDList {
		plugin, ok := pluginByID[pluginID]
		if !ok {
			continue
		}

		cli, err := openMCPClient(ctx, plugin.MCPURL, plugin.MCPProtocol)
		if err != nil {
			closeMCPClients(closers)
			return nil, nil, fmt.Errorf("connect mcp plugin %s failed: %w", pluginID, err)
		}

		pluginTools, err := mcpTool.GetTools(ctx, &mcpTool.Config{
			Cli:          cli,
			ToolNameList: toolNamesByPlugin[pluginID],
		})
		if err != nil {
			_ = cli.Close()
			closeMCPClients(closers)
			return nil, nil, fmt.Errorf("load mcp plugin %s tools failed: %w", pluginID, err)
		}

		// Use plugin-level timeout for all tools in this plugin
		timeoutMS := plugin.TimeoutMS
		if timeoutMS <= 0 {
			timeoutMS = defaultPluginTimeoutMS
		}

		// Wrap each tool with timeout
		for _, tool := range pluginTools {
			wrappedTool := newMCPToolWithTimeout(tool, timeoutMS)
			runtimeTools = append(runtimeTools, wrappedTool)
		}
		closers = append(closers, cli)
	}

	cleanup := func() {
		closeMCPClients(closers)
	}
	return runtimeTools, cleanup, nil
}

func closeMCPClients(clients []*client.Client) {
	for _, cli := range clients {
		if cli == nil {
			continue
		}
		if err := cli.Close(); err != nil {
			log.Printf("close mcp client failed err=%v", err)
		}
	}
}
