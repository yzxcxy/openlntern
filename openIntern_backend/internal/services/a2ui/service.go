package a2ui

import (
	"errors"
	"openIntern/internal/dao"
	"openIntern/internal/models"
)

type A2UIService struct{}

var A2UI = new(A2UIService)

// CreateA2UI 创建 A2UI
func (s *A2UIService) CreateA2UI(a2ui *models.A2UI) error {
	return dao.A2UI.Create(a2ui)
}

// GetA2UIByID 根据 A2UIID 获取
func (s *A2UIService) GetA2UIByID(userID, id string) (*models.A2UI, error) {
	return dao.A2UI.GetByUserIDAndID(userID, id)
}

// UpdateA2UI 更新 A2UI
func (s *A2UIService) UpdateA2UI(userID, id string, updates map[string]interface{}) error {
	rowsAffected, err := dao.A2UI.UpdateByUserIDAndID(userID, id, updates)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("a2ui not found")
	}
	return nil
}

// DeleteA2UI 删除 A2UI
func (s *A2UIService) DeleteA2UI(userID, id string) error {
	rowsAffected, err := dao.A2UI.DeleteByUserIDAndID(userID, id)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("a2ui not found")
	}
	return nil
}

// ListA2UIs 获取 A2UI 列表（分页）
func (s *A2UIService) ListA2UIs(userID string, page, pageSize int, keyword string) ([]models.A2UI, int64, error) {
	return dao.A2UI.ListByUserID(userID, page, pageSize, dao.A2UIListFilter{Keyword: keyword})
}
