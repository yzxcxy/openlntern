package dao

import (
	"strings"

	"openIntern/internal/database"
	"openIntern/internal/models"
)

type A2UIListFilter struct {
	Keyword string
}

type A2UIDAO struct{}

var A2UI = new(A2UIDAO)

func (d *A2UIDAO) Create(item *models.A2UI) error {
	return database.DB.Create(item).Error
}

func (d *A2UIDAO) GetByID(id string) (*models.A2UI, error) {
	var item models.A2UI
	if err := database.DB.Where("a2ui_id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *A2UIDAO) UpdateByID(id string, updates map[string]any) (int64, error) {
	result := database.DB.Model(&models.A2UI{}).Where("a2ui_id = ?", id).Updates(updates)
	return result.RowsAffected, result.Error
}

func (d *A2UIDAO) DeleteByID(id string) (int64, error) {
	result := database.DB.Where("a2ui_id = ?", id).Delete(&models.A2UI{})
	return result.RowsAffected, result.Error
}

func (d *A2UIDAO) List(page, pageSize int, filter A2UIListFilter) ([]models.A2UI, int64, error) {
	_, pageSize, offset := normalizePagination(page, pageSize, 10)

	query := database.DB.Model(&models.A2UI{})
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		pattern := "%" + keyword + "%"
		query = query.Where("(name LIKE ? OR description LIKE ?)", pattern, pattern)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []models.A2UI
	if err := query.Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}
