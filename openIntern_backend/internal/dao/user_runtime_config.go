package dao

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"openIntern/internal/database"
	"openIntern/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserRuntimeConfigDAO stores per-user runtime config overrides.
type UserRuntimeConfigDAO struct{}

// UserRuntimeConfig is the shared DAO singleton.
var UserRuntimeConfig = new(UserRuntimeConfigDAO)

func (d *UserRuntimeConfigDAO) GetByUserIDAndKey(userID, configKey string) (*models.UserRuntimeConfig, error) {
	userID = strings.TrimSpace(userID)
	configKey = strings.TrimSpace(configKey)
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if configKey == "" {
		return nil, fmt.Errorf("config_key is required")
	}

	var item models.UserRuntimeConfig
	if err := database.DB.Where("user_id = ? AND config_key = ?", userID, configKey).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (d *UserRuntimeConfigDAO) ListByUserID(userID string, configKeys []string) ([]models.UserRuntimeConfig, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	keys := normalizeUserRuntimeConfigKeys(configKeys)
	if len(keys) == 0 {
		return []models.UserRuntimeConfig{}, nil
	}

	var items []models.UserRuntimeConfig
	if err := database.DB.Where("user_id = ? AND config_key IN ?", userID, keys).Order("config_key ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (d *UserRuntimeConfigDAO) Upsert(userID, configKey string, configValue []byte) (*models.UserRuntimeConfig, error) {
	userID = strings.TrimSpace(userID)
	configKey = strings.TrimSpace(configKey)
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if configKey == "" {
		return nil, fmt.Errorf("config_key is required")
	}
	if len(strings.TrimSpace(string(configValue))) == 0 {
		return nil, fmt.Errorf("config_value is required")
	}
	if !json.Valid(configValue) {
		return nil, fmt.Errorf("config_value must be valid JSON")
	}

	item := models.UserRuntimeConfig{
		UserID:      userID,
		ConfigKey:   configKey,
		ConfigValue: json.RawMessage(configValue),
	}
	if err := database.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "config_key"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"config_value", "updated_at"}),
	}).Create(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func normalizeUserRuntimeConfigKeys(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
