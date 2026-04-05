package sandbox

import (
	"context"
	"time"
)

const (
	StatusProvisioning = "provisioning"
	StatusReady        = "ready"
	StatusFailed       = "failed"
	StatusRecycling    = "recycling"

	ProviderDocker = "docker"
	ProviderSCF    = "scf"
)

// Instance 是服务层使用的 sandbox 实例视图。
type Instance struct {
	UserID         string
	Provider       string
	Status         string
	InstanceID     string
	Endpoint       string
	LastActiveAt   time.Time
	LeaseExpiresAt time.Time
	LastError      string
}

// CreateRequest 描述一次用户级实例创建请求。
type CreateRequest struct {
	UserID string
}

// CreateResult 是 provider 创建或认领实例后的最小返回结果。
type CreateResult struct {
	InstanceID string
	Endpoint   string
}

// Store 负责 sandbox 实例元数据的持久化。
type Store interface {
	GetByUserID(userID string) (*Instance, error)
	UpsertProvisioning(userID, provider string, leaseExpiresAt time.Time) (*Instance, error)
	MarkReady(userID string, updates map[string]any) error
	MarkFailed(userID, message string) error
	MarkRecyclingIfExpired(userID string, now time.Time) (bool, error)
	TouchReady(userID string, lastActiveAt, leaseExpiresAt time.Time) error
	DeleteByUserID(userID string) error
	ListExpiredReady(now time.Time, limit int) ([]Instance, error)
}

// Provider 屏蔽本地 Docker 和未来 SCF 的资源创建差异。
type Provider interface {
	Name() string
	Create(ctx context.Context, req CreateRequest) (*CreateResult, error)
	FindExisting(ctx context.Context, userID string) (*CreateResult, error)
	Destroy(ctx context.Context, instance Instance) error
	HealthCheck(ctx context.Context, instance Instance) error
}
