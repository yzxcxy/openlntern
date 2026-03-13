package memory

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
	memorySyncLockRedisKeyPrefix = "openintern:memory-sync:lock:"
	memorySyncLockAcquireTimeout = 2 * time.Second
	memorySyncLockReleaseTimeout = 2 * time.Second
	memorySyncLockExtraTTL       = 30 * time.Second
)

var releaseMemorySyncLockScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`

// memorySyncLock represents one thread-scoped Redis lease for long-term memory sync.
type memorySyncLock struct {
	key   string
	token string
}

// tryAcquireMemorySyncLock acquires a Redis lease per thread so only one instance processes it at a time.
func tryAcquireMemorySyncLock(threadID string) (*memorySyncLock, bool, error) {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return nil, false, errors.New("thread_id is required")
	}

	redisClient := database.GetRedis()
	if redisClient == nil {
		return nil, false, errors.New("memory sync requires redis")
	}

	token, err := newMemorySyncLockToken()
	if err != nil {
		return nil, false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), memorySyncLockAcquireTimeout)
	defer cancel()

	key := memorySyncLockRedisKeyPrefix + threadID
	acquired, err := redisClient.SetNX(ctx, key, token, currentMemorySyncLockTTL()).Result()
	if err != nil || !acquired {
		return nil, acquired, err
	}
	return &memorySyncLock{key: key, token: token}, true, nil
}

// Release deletes the Redis lease only when the same worker token still owns it.
func (l *memorySyncLock) Release() error {
	if l == nil {
		return nil
	}

	redisClient := database.GetRedis()
	if redisClient == nil {
		return errors.New("memory sync requires redis")
	}

	ctx, cancel := context.WithTimeout(context.Background(), memorySyncLockReleaseTimeout)
	defer cancel()

	return redisClient.Eval(ctx, releaseMemorySyncLockScript, []string{l.key}, l.token).Err()
}

// currentMemorySyncLockTTL keeps the Redis lease slightly longer than one sync attempt timeout.
func currentMemorySyncLockTTL() time.Duration {
	ttl := memorySyncTimeout + memorySyncLockExtraTTL
	if ttl <= 0 {
		return defaultMemorySyncTimeout + memorySyncLockExtraTTL
	}
	return ttl
}

// newMemorySyncLockToken generates a random token so lock release can verify ownership.
func newMemorySyncLockToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate memory sync lock token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
