package dao

import (
	"errors"
	"strings"
	"time"

	"openIntern/internal/database"
	"openIntern/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrSandboxInstanceNotFound = errors.New("sandbox instance not found")

// SandboxInstanceDAO 提供用户级 sandbox 实例持久化能力。
type SandboxInstanceDAO struct{}

// SandboxInstance 是共享 DAO 单例。
var SandboxInstance = new(SandboxInstanceDAO)

func (d *SandboxInstanceDAO) GetByUserID(userID string) (*models.SandboxInstance, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrSandboxInstanceNotFound
	}

	var item models.SandboxInstance
	if err := database.DB.Where("user_id = ?", userID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSandboxInstanceNotFound
		}
		return nil, err
	}
	return &item, nil
}

// UpsertProvisioning 在创建开始前占位，避免同一用户出现多条实例记录。
func (d *SandboxInstanceDAO) UpsertProvisioning(userID, provider string, leaseExpiresAt time.Time) (*models.SandboxInstance, error) {
	userID = strings.TrimSpace(userID)
	provider = strings.TrimSpace(provider)
	if userID == "" {
		return nil, ErrSandboxInstanceNotFound
	}

	item := &models.SandboxInstance{
		UserID:         userID,
		Provider:       provider,
		Status:         "provisioning",
		InstanceID:     "",
		Endpoint:       "",
		LastActiveAt:   time.Now(),
		LeaseExpiresAt: leaseExpiresAt,
		LastError:      "",
	}

	if err := database.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"provider":         provider,
			"status":           "provisioning",
			"instance_id":      "",
			"endpoint":         "",
			"lease_expires_at": leaseExpiresAt,
			"last_error":       "",
		}),
	}).Create(item).Error; err != nil {
		return nil, err
	}

	return d.GetByUserID(userID)
}

func (d *SandboxInstanceDAO) MarkReady(userID string, updates map[string]any) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrSandboxInstanceNotFound
	}

	payload := map[string]any{
		"status":     "ready",
		"last_error": "",
	}
	for key, value := range updates {
		payload[key] = value
	}

	return database.DB.Model(&models.SandboxInstance{}).
		Where("user_id = ?", userID).
		Updates(payload).Error
}

func (d *SandboxInstanceDAO) MarkFailed(userID, message string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrSandboxInstanceNotFound
	}

	return database.DB.Model(&models.SandboxInstance{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"status":     "failed",
			"last_error": strings.TrimSpace(message),
		}).Error
}

// MarkRecyclingIfExpired 仅在实例仍然处于 ready 且租约已过期时推进回收状态。
func (d *SandboxInstanceDAO) MarkRecyclingIfExpired(userID string, now time.Time) (bool, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false, ErrSandboxInstanceNotFound
	}

	result := database.DB.Model(&models.SandboxInstance{}).
		Where("user_id = ? AND status = ? AND lease_expires_at <= ?", userID, "ready", now).
		Update("status", "recycling")
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (d *SandboxInstanceDAO) TouchReady(userID string, lastActiveAt, leaseExpiresAt time.Time) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrSandboxInstanceNotFound
	}

	return database.DB.Model(&models.SandboxInstance{}).
		Where("user_id = ? AND status = ?", userID, "ready").
		Updates(map[string]any{
			"last_active_at":   lastActiveAt,
			"lease_expires_at": leaseExpiresAt,
		}).Error
}

func (d *SandboxInstanceDAO) DeleteByUserID(userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrSandboxInstanceNotFound
	}

	return database.DB.Where("user_id = ?", userID).Delete(&models.SandboxInstance{}).Error
}

func (d *SandboxInstanceDAO) ListExpiredReady(now time.Time, limit int) ([]models.SandboxInstance, error) {
	if limit <= 0 {
		limit = 20
	}

	var items []models.SandboxInstance
	if err := database.DB.
		Where("status = ? AND lease_expires_at <= ?", "ready", now).
		Order("lease_expires_at ASC").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
