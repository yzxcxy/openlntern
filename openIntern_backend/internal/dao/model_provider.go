package dao

import (
	"strings"

	"openIntern/internal/database"
	"openIntern/internal/models"
)

type ModelProviderListFilter struct {
	UserID  string
	Keyword string
}

type ModelProviderDAO struct{}

var ModelProvider = new(ModelProviderDAO)

func (d *ModelProviderDAO) Create(item *models.ModelProvider) error {
	return database.DB.Create(item).Error
}

func (d *ModelProviderDAO) GetByUserIDAndProviderID(userID string, providerID string) (*models.ModelProvider, error) {
	var item models.ModelProvider
	if err := database.DB.Where("user_id = ? AND provider_id = ?", strings.TrimSpace(userID), providerID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *ModelProviderDAO) UpdateByUserIDAndProviderID(userID string, providerID string, updates map[string]any) (int64, error) {
	result := database.DB.Model(&models.ModelProvider{}).Where("user_id = ? AND provider_id = ?", strings.TrimSpace(userID), providerID).Updates(updates)
	return result.RowsAffected, result.Error
}

func (d *ModelProviderDAO) CountModelsByUserIDAndProviderID(userID string, providerID string) (int64, error) {
	var count int64
	if err := database.DB.Model(&models.ModelCatalog{}).Where("user_id = ? AND provider_id = ?", strings.TrimSpace(userID), providerID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (d *ModelProviderDAO) DeleteByUserIDAndProviderID(userID string, providerID string) (int64, error) {
	result := database.DB.Where("user_id = ? AND provider_id = ?", strings.TrimSpace(userID), providerID).Delete(&models.ModelProvider{})
	return result.RowsAffected, result.Error
}

func (d *ModelProviderDAO) List(page, pageSize int, filter ModelProviderListFilter) ([]models.ModelProvider, int64, error) {
	_, pageSize, offset := normalizePagination(page, pageSize, 10)

	query := database.DB.Model(&models.ModelProvider{}).Where("user_id = ?", strings.TrimSpace(filter.UserID))
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		pattern := "%" + keyword + "%"
		query = query.Where("(name LIKE ? OR api_type LIKE ?)", pattern, pattern)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []models.ModelProvider
	if err := query.Order("updated_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (d *ModelProviderDAO) LoadByUserIDAndProviderIDs(userID string, providerIDs []string) (map[string]*models.ModelProvider, error) {
	result := make(map[string]*models.ModelProvider, len(providerIDs))
	if len(providerIDs) == 0 {
		return result, nil
	}

	var items []models.ModelProvider
	if err := database.DB.Where("user_id = ? AND provider_id IN ?", strings.TrimSpace(userID), providerIDs).Find(&items).Error; err != nil {
		return nil, err
	}

	for i := range items {
		item := items[i]
		result[item.ProviderID] = &item
	}

	return result, nil
}
