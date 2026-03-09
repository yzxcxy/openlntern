package plugin

import (
	"errors"

	"openIntern/internal/database"
)

// ensureSyncQueuesReady 在需要同步队列时校验 Redis 可用性。
func (s *PluginService) ensureSyncQueuesReady(requireMCPSync bool, requireOpenVikingSync bool) error {
	if !requireMCPSync && !requireOpenVikingSync {
		return nil
	}
	if database.GetRedis() == nil {
		return errors.New("plugin sync queues require redis")
	}
	return nil
}
