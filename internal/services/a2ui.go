package services

import (
	"errors"
	"openIntern/internal/database"
	"openIntern/internal/models"
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
func (s *A2UIService) UpdateA2UI(id string, updates map[string]interface{}, operatorID string) error {
	// 1. 获取目标 A2UI
	var a2ui models.A2UI
	if err := database.DB.Where("a2ui_id = ?", id).First(&a2ui).Error; err != nil {
		return err
	}

	// 2. 如果是官方 A2UI，检查权限
	if a2ui.Type == models.A2UITypeOfficial {
		if operatorID == "" {
			return errors.New("permission denied: authentication required for official a2ui")
		}
		operator, err := User.GetUserByUserID(operatorID)
		if err != nil {
			return errors.New("permission denied: invalid user")
		}
		if operator.Role != models.RoleAdmin {
			return errors.New("permission denied: only admin can update official a2ui")
		}
	}

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
func (s *A2UIService) DeleteA2UI(id string, operatorID string) error {
	// 1. 获取目标 A2UI
	var a2ui models.A2UI
	if err := database.DB.Where("a2ui_id = ?", id).First(&a2ui).Error; err != nil {
		return err
	}

	// 2. 如果是官方 A2UI，检查权限
	if a2ui.Type == models.A2UITypeOfficial {
		if operatorID == "" {
			return errors.New("permission denied: authentication required for official a2ui")
		}
		operator, err := User.GetUserByUserID(operatorID)
		if err != nil {
			return errors.New("permission denied: invalid user")
		}
		if operator.Role != models.RoleAdmin {
			return errors.New("permission denied: only admin can delete official a2ui")
		}
	}

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
func (s *A2UIService) ListA2UIs(page, pageSize int, userID uint) ([]models.A2UI, int64, error) {
	var a2uis []models.A2UI
	var total int64

	offset := (page - 1) * pageSize

	db := database.DB.Model(&models.A2UI{})
	if userID != 0 {
		db = db.Where("user_id = ?", userID)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := db.Offset(offset).Limit(pageSize).Find(&a2uis).Error; err != nil {
		return nil, 0, err
	}

	return a2uis, total, nil
}

// ListOfficialA2UIs 获取官方 A2UI 列表
func (s *A2UIService) ListOfficialA2UIs(page, pageSize int) ([]models.A2UI, int64, error) {
	var a2uis []models.A2UI
	var total int64
	offset := (page - 1) * pageSize

	db := database.DB.Model(&models.A2UI{}).Where("type = ?", models.A2UITypeOfficial)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := db.Offset(offset).Limit(pageSize).Find(&a2uis).Error; err != nil {
		return nil, 0, err
	}

	return a2uis, total, nil
}

// ListCustomA2UIs 获取自定义 A2UI 列表
func (s *A2UIService) ListCustomA2UIs(page, pageSize int, userID uint) ([]models.A2UI, int64, error) {
	var a2uis []models.A2UI
	var total int64
	offset := (page - 1) * pageSize

	db := database.DB.Model(&models.A2UI{}).Where("type = ?", models.A2UITypeCustom)

	if userID != 0 {
		db = db.Where("user_id = ?", userID)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := db.Offset(offset).Limit(pageSize).Find(&a2uis).Error; err != nil {
		return nil, 0, err
	}

	return a2uis, total, nil
}
