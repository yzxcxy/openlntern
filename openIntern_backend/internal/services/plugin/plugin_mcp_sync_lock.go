package plugin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"openIntern/internal/database"
)

const (
	mcpSyncLockRedisKeyPrefix = "openintern:plugin:mcp:sync-lock:"
	mcpSyncLockAcquireTimeout = 2 * time.Second
	mcpSyncLockReleaseTimeout = 2 * time.Second
	mcpSyncLockExtraTTL       = 30 * time.Second
)

var releaseMCPSyncLockScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`

// mcpSyncLock 表示插件级 MCP 同步租约，保证同一插件同一时刻只会被一个实例同步。
type mcpSyncLock struct {
	key   string
	token string
}

func tryAcquireMCPPluginSyncLock(pluginID string) (*mcpSyncLock, bool, error) {
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return nil, false, errors.New("plugin_id is required")
	}

	redisClient := database.GetRedis()
	if redisClient == nil {
		return nil, false, errors.New("mcp sync requires redis")
	}

	token, err := newMCPSyncLockToken()
	if err != nil {
		return nil, false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), mcpSyncLockAcquireTimeout)
	defer cancel()

	key := mcpSyncLockRedisKeyPrefix + pluginID
	acquired, err := redisClient.SetNX(ctx, key, token, currentMCPSyncLockTTL()).Result()
	if err != nil || !acquired {
		return nil, acquired, err
	}
	return &mcpSyncLock{key: key, token: token}, true, nil
}

func (l *mcpSyncLock) Release() error {
	if l == nil {
		return nil
	}

	redisClient := database.GetRedis()
	if redisClient == nil {
		return errors.New("mcp sync requires redis")
	}

	ctx, cancel := context.WithTimeout(context.Background(), mcpSyncLockReleaseTimeout)
	defer cancel()

	return redisClient.Eval(ctx, releaseMCPSyncLockScript, []string{l.key}, l.token).Err()
}

func currentMCPSyncLockTTL() time.Duration {
	ttl := mcpSyncTimeout + mcpSyncLockExtraTTL
	if ttl <= 0 {
		return defaultMCPSyncTimeout + mcpSyncLockExtraTTL
	}
	return ttl
}

func newMCPSyncLockToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate mcp sync lock token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
