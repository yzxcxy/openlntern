package services

import (
	"errors"
	"openIntern/internal/database"
	"openIntern/internal/models"
	"strings"
)

type A2UIService struct{}

var A2UI = new(A2UIService)

// CreateA2UI 创建 A2UI
func (s *A2UIService) CreateA2UI(a2ui *models.A2UI) error {
	return database.DB.Create(a2ui).Error
}

// GetA2UIByID 根据 A2UIID 获取
func (s *A2UIService) GetA2UIByID(id string) (*models.A2UI, error) {
	var a2ui models.A2UI
	err := database.DB.Where("a2ui_id = ?", id).First(&a2ui).Error
	if err != nil {
		return nil, err
	}
	return &a2ui, nil
}

// UpdateA2UI 更新 A2UI
func (s *A2UIService) UpdateA2UI(id string, updates map[string]interface{}) error {
	result := database.DB.Model(&models.A2UI{}).Where("a2ui_id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("a2ui not found")
	}
	return nil
}

// DeleteA2UI 删除 A2UI
func (s *A2UIService) DeleteA2UI(id string) error {
	result := database.DB.Where("a2ui_id = ?", id).Delete(&models.A2UI{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("a2ui not found")
	}
	return nil
}

// ListA2UIs 获取 A2UI 列表（分页）
func (s *A2UIService) ListA2UIs(page, pageSize int, keyword string) ([]models.A2UI, int64, error) {
	var a2uis []models.A2UI
	var total int64

	offset := (page - 1) * pageSize

	db := database.DB.Model(&models.A2UI{})
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		pattern := "%" + keyword + "%"
		db = db.Where("(name LIKE ? OR description LIKE ?)", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := db.Offset(offset).Limit(pageSize).Find(&a2uis).Error; err != nil {
		return nil, 0, err
	}

	return a2uis, total, nil
}
