package sandbox

import (
	"context"
	"fmt"
	"strings"
	"time"

	"openIntern/internal/config"
)

// BuildProvider 根据配置构造本地 sandbox provider。
func BuildProvider(cfg config.SandboxConfig) (Provider, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	switch provider {
	case ProviderDocker:
		return NewDockerProvider(cfg)
	case ProviderSCF:
		return nil, fmt.Errorf("sandbox provider %s is not implemented", ProviderSCF)
	case "":
		return nil, fmt.Errorf("tools.sandbox.provider is required")
	default:
		return nil, fmt.Errorf("unsupported sandbox provider: %s", provider)
	}
}

// healthcheckContext 为探活创建独立的短超时上下文。
func healthcheckContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if ctx == nil {
		return context.WithTimeout(context.Background(), timeout)
	}
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}
