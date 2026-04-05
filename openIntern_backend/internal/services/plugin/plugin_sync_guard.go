package plugin

import (
	"errors"

	"openIntern/internal/database"
)

// ensureSyncQueuesReady 在需要 MCP 同步队列时校验 Redis 可用性。
func (s *PluginService) ensureSyncQueuesReady(requireMCPSync bool) error {
	if !requireMCPSync {
		return nil
	}
	if database.GetRedis() == nil {
		return errors.New("plugin sync queues require redis")
	}
	return nil
}
