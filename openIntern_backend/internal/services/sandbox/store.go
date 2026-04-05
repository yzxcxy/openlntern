package sandbox

import (
	"time"

	"openIntern/internal/dao"
)

type daoStore struct{}

// NewStore 返回基于 DAO 的 sandbox 元数据存储实现。
func NewStore() Store {
	return &daoStore{}
}

func (s *daoStore) GetByUserID(userID string) (*Instance, error) {
	item, err := dao.SandboxInstance.GetByUserID(userID)
	if err != nil {
		return nil, err
	}
	return fromModel(item), nil
}

func (s *daoStore) UpsertProvisioning(userID, provider string, leaseExpiresAt time.Time) (*Instance, error) {
	item, err := dao.SandboxInstance.UpsertProvisioning(userID, provider, leaseExpiresAt)
	if err != nil {
		return nil, err
	}
	return fromModel(item), nil
}

func (s *daoStore) MarkReady(userID string, updates map[string]any) error {
	return dao.SandboxInstance.MarkReady(userID, updates)
}

func (s *daoStore) MarkFailed(userID, message string) error {
	return dao.SandboxInstance.MarkFailed(userID, message)
}

func (s *daoStore) MarkRecyclingIfExpired(userID string, now time.Time) (bool, error) {
	return dao.SandboxInstance.MarkRecyclingIfExpired(userID, now)
}

func (s *daoStore) TouchReady(userID string, lastActiveAt, leaseExpiresAt time.Time) error {
	return dao.SandboxInstance.TouchReady(userID, lastActiveAt, leaseExpiresAt)
}

func (s *daoStore) DeleteByUserID(userID string) error {
	return dao.SandboxInstance.DeleteByUserID(userID)
}

func (s *daoStore) ListExpiredReady(now time.Time, limit int) ([]Instance, error) {
	items, err := dao.SandboxInstance.ListExpiredReady(now, limit)
	if err != nil {
		return nil, err
	}

	result := make([]Instance, 0, len(items))
	for i := range items {
		result = append(result, *fromModel(&items[i]))
	}
	return result, nil
}
