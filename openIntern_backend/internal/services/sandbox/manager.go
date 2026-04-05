package sandbox

import (
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"openIntern/internal/config"
	"openIntern/internal/dao"
	"openIntern/internal/models"
)

type Service struct {
	mu              sync.RWMutex
	store           Store
	provider        Provider
	client          *Client
	idleTTL         time.Duration
	createTimeout   time.Duration
	recycleInterval time.Duration
	stopCh          chan struct{}
	userLocks       sync.Map
}

// Lifecycle 是全局 sandbox 生命周期服务实例。
var Lifecycle = new(Service)

// Init 基于配置初始化 sandbox 生命周期管理与后台回收协程。
func Init(cfg config.SandboxConfig) (func(), error) {
	return Lifecycle.Init(cfg)
}

func (s *Service) Init(cfg config.SandboxConfig) (func(), error) {
	if cfg.Enabled == nil || !*cfg.Enabled {
		return nil, errors.New("tools.sandbox.enabled must be true")
	}

	provider, err := BuildProvider(cfg)
	if err != nil {
		return nil, err
	}

	idleTTL := time.Duration(cfg.IdleTTLSeconds) * time.Second
	if idleTTL <= 0 {
		idleTTL = 30 * time.Minute
	}
	createTimeout := time.Duration(cfg.CreateTimeoutSeconds) * time.Second
	if createTimeout <= 0 {
		createTimeout = 30 * time.Second
	}
	recycleInterval := time.Duration(cfg.RecycleIntervalSeconds) * time.Second
	if recycleInterval <= 0 {
		recycleInterval = 30 * time.Second
	}

	s.mu.Lock()
	s.store = NewStore()
	s.provider = provider
	s.client = NewClient()
	s.idleTTL = idleTTL
	s.createTimeout = createTimeout
	s.recycleInterval = recycleInterval
	if s.stopCh != nil {
		close(s.stopCh)
	}
	s.stopCh = make(chan struct{})
	stopCh := s.stopCh
	s.mu.Unlock()

	go s.runRecycler(stopCh)
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.stopCh == stopCh {
			close(stopCh)
			s.stopCh = nil
		}
	}, nil
}

func (s *Service) Client() *Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client
}

func (s *Service) GetOrCreate(ctx context.Context, userID string) (*Instance, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("user_id is required")
	}

	lock := s.userLock(userID)
	lock.Lock()
	defer lock.Unlock()

	return s.getOrCreateLocked(ctx, userID)
}

func (s *Service) Touch(ctx context.Context, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("user_id is required")
	}

	now := time.Now()
	return s.store.TouchReady(userID, now, now.Add(s.idleTTL))
}

func (s *Service) Destroy(ctx context.Context, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("user_id is required")
	}

	lock := s.userLock(userID)
	lock.Lock()
	defer lock.Unlock()

	item, err := s.store.GetByUserID(userID)
	if err != nil && !errors.Is(err, dao.ErrSandboxInstanceNotFound) {
		return err
	}

	if item != nil {
		if err := s.provider.Destroy(ctx, *item); err != nil {
			_ = s.store.MarkFailed(userID, err.Error())
			return err
		}
		return s.store.DeleteByUserID(userID)
	}

	existing, err := s.provider.FindExisting(ctx, userID)
	if err != nil {
		return err
	}
	if existing == nil {
		return nil
	}
	return s.provider.Destroy(ctx, Instance{
		UserID:     userID,
		Provider:   s.provider.Name(),
		InstanceID: existing.InstanceID,
		Endpoint:   existing.Endpoint,
	})
}

func (s *Service) RecycleIdle(ctx context.Context, limit int) (int, error) {
	items, err := s.store.ListExpiredReady(time.Now(), limit)
	if err != nil {
		return 0, err
	}

	recycled := 0
	for _, item := range items {
		lock := s.userLock(item.UserID)
		lock.Lock()

		ok, markErr := s.store.MarkRecyclingIfExpired(item.UserID, time.Now())
		if markErr != nil {
			lock.Unlock()
			return recycled, markErr
		}
		if !ok {
			lock.Unlock()
			continue
		}

		if destroyErr := s.provider.Destroy(ctx, item); destroyErr != nil {
			_ = s.store.MarkFailed(item.UserID, destroyErr.Error())
			lock.Unlock()
			continue
		}
		if deleteErr := s.store.DeleteByUserID(item.UserID); deleteErr != nil {
			lock.Unlock()
			return recycled, deleteErr
		}
		recycled++
		lock.Unlock()
	}
	return recycled, nil
}

func (s *Service) getOrCreateLocked(ctx context.Context, userID string) (*Instance, error) {
	if item, err := s.store.GetByUserID(userID); err == nil {
		if reusable, reuseErr := s.reuseReady(ctx, *item); reuseErr != nil {
			return nil, reuseErr
		} else if reusable != nil {
			return reusable, nil
		}
	} else if !errors.Is(err, dao.ErrSandboxInstanceNotFound) {
		return nil, err
	}

	if claimed, err := s.claimExisting(ctx, userID); err != nil {
		return nil, err
	} else if claimed != nil {
		return claimed, nil
	}

	leaseExpiresAt := time.Now().Add(s.idleTTL)
	if _, err := s.store.UpsertProvisioning(userID, s.provider.Name(), leaseExpiresAt); err != nil {
		return nil, err
	}

	createCtx := ctx
	cancel := func() {}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		createCtx, cancel = context.WithTimeout(ctx, s.createTimeout)
	}
	defer cancel()

	result, err := s.provider.Create(createCtx, CreateRequest{UserID: userID})
	if err != nil {
		_ = s.store.MarkFailed(userID, err.Error())
		return nil, err
	}

	instance := &Instance{
		UserID:         userID,
		Provider:       s.provider.Name(),
		Status:         StatusReady,
		InstanceID:     result.InstanceID,
		Endpoint:       result.Endpoint,
		LastActiveAt:   time.Now(),
		LeaseExpiresAt: time.Now().Add(s.idleTTL),
	}

	if err := s.provider.HealthCheck(createCtx, *instance); err != nil {
		_ = s.store.MarkFailed(userID, err.Error())
		return nil, err
	}
	if err := s.store.MarkReady(userID, readyUpdateMap(*instance)); err != nil {
		return nil, err
	}
	return instance, nil
}

func (s *Service) claimExisting(ctx context.Context, userID string) (*Instance, error) {
	result, err := s.provider.FindExisting(ctx, userID)
	if err != nil || result == nil {
		return nil, err
	}

	now := time.Now()
	instance := Instance{
		UserID:         userID,
		Provider:       s.provider.Name(),
		Status:         StatusReady,
		InstanceID:     result.InstanceID,
		Endpoint:       result.Endpoint,
		LastActiveAt:   now,
		LeaseExpiresAt: now.Add(s.idleTTL),
	}
	if err := s.provider.HealthCheck(ctx, instance); err != nil {
		return nil, err
	}
	if _, err := s.store.UpsertProvisioning(userID, s.provider.Name(), instance.LeaseExpiresAt); err != nil {
		return nil, err
	}
	if err := s.store.MarkReady(userID, readyUpdateMap(instance)); err != nil {
		return nil, err
	}
	return &instance, nil
}

func (s *Service) reuseReady(ctx context.Context, item Instance) (*Instance, error) {
	if item.Status != StatusReady || item.LeaseExpiresAt.Before(time.Now()) {
		return nil, nil
	}
	if strings.TrimSpace(item.Endpoint) == "" || strings.TrimSpace(item.InstanceID) == "" {
		return nil, nil
	}
	if err := s.provider.HealthCheck(ctx, item); err != nil {
		return nil, nil
	}

	now := time.Now()
	item.LastActiveAt = now
	item.LeaseExpiresAt = now.Add(s.idleTTL)
	if err := s.store.TouchReady(item.UserID, item.LastActiveAt, item.LeaseExpiresAt); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) runRecycler(stopCh chan struct{}) {
	ticker := time.NewTicker(s.recycleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if _, err := s.RecycleIdle(context.Background(), 20); err != nil {
				log.Printf("sandbox recycler failed err=%v", err)
			}
		case <-stopCh:
			return
		}
	}
}

func (s *Service) userLock(userID string) *sync.Mutex {
	lock, _ := s.userLocks.LoadOrStore(userID, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func fromModel(item *models.SandboxInstance) *Instance {
	if item == nil {
		return nil
	}
	return &Instance{
		UserID:         item.UserID,
		Provider:       item.Provider,
		Status:         item.Status,
		InstanceID:     item.InstanceID,
		Endpoint:       item.Endpoint,
		LastActiveAt:   item.LastActiveAt,
		LeaseExpiresAt: item.LeaseExpiresAt,
		LastError:      item.LastError,
	}
}

func readyUpdateMap(item Instance) map[string]any {
	return map[string]any{
		"provider":         item.Provider,
		"status":           StatusReady,
		"instance_id":      item.InstanceID,
		"endpoint":         item.Endpoint,
		"last_active_at":   item.LastActiveAt,
		"lease_expires_at": item.LeaseExpiresAt,
	}
}
