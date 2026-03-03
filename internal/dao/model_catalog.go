package dao

import (
	"errors"
	"strings"

	"openIntern/internal/database"
	"openIntern/internal/models"

	"gorm.io/gorm"
)

type ModelCatalogListFilter struct {
	Keyword    string
	ProviderID string
}

type ModelCatalogDAO struct{}

type DefaultModelConfigDAO struct{}

var ModelCatalog = new(ModelCatalogDAO)
var DefaultModelConfig = new(DefaultModelConfigDAO)

func (d *ModelCatalogDAO) Create(item *models.ModelCatalog) error {
	return database.DB.Create(item).Error
}

func (d *ModelCatalogDAO) GetByModelID(modelID string) (*models.ModelCatalog, error) {
	var item models.ModelCatalog
	if err := database.DB.Where("model_id = ?", modelID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *ModelCatalogDAO) UpdateByModelID(modelID string, updates map[string]any) (int64, error) {
	result := database.DB.Model(&models.ModelCatalog{}).Where("model_id = ?", modelID).Updates(updates)
	return result.RowsAffected, result.Error
}

func (d *ModelCatalogDAO) DeleteByModelID(modelID string) (int64, error) {
	result := database.DB.Where("model_id = ?", modelID).Delete(&models.ModelCatalog{})
	return result.RowsAffected, result.Error
}

func (d *ModelCatalogDAO) List(page, pageSize int, filter ModelCatalogListFilter) ([]models.ModelCatalog, int64, error) {
	_, pageSize, offset := normalizePagination(page, pageSize, 10)

	query := database.DB.Model(&models.ModelCatalog{})
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		pattern := "%" + keyword + "%"
		query = query.Where("(name LIKE ? OR model_key LIKE ?)", pattern, pattern)
	}
	if providerID := strings.TrimSpace(filter.ProviderID); providerID != "" {
		query = query.Where("provider_id = ?", providerID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []models.ModelCatalog
	if err := query.Order("sort ASC").Order("updated_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (d *ModelCatalogDAO) ListEnabled() ([]models.ModelCatalog, error) {
	var items []models.ModelCatalog
	if err := database.DB.Where("enabled = ?", true).Order("sort ASC").Order("updated_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (d *DefaultModelConfigDAO) GetByConfigKey(configKey string) (*models.DefaultModelConfig, error) {
	var item models.DefaultModelConfig
	if err := database.DB.Where("config_key = ?", configKey).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (d *DefaultModelConfigDAO) UpsertByConfigKey(configKey, modelID string) (*models.DefaultModelConfig, error) {
	item, err := d.GetByConfigKey(configKey)
	if err != nil {
		return nil, err
	}
	if item == nil {
		created := models.DefaultModelConfig{
			ConfigKey: configKey,
			ModelID:   modelID,
		}
		if err := database.DB.Create(&created).Error; err != nil {
			return nil, err
		}
		return &created, nil
	}

	item.ModelID = modelID
	if err := database.DB.Save(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}
