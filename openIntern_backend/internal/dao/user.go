package dao

import (
	"openIntern/internal/database"
	"openIntern/internal/models"
)

type UserDAO struct{}

var User = new(UserDAO)

func (d *UserDAO) ExistsByUsernameOrEmail(username, email string) (bool, error) {
	var count int64
	if err := database.DB.Model(&models.User{}).
		Where("username = ? OR email = ?", username, email).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (d *UserDAO) Create(user *models.User) error {
	return database.DB.Create(user).Error
}

func (d *UserDAO) GetByUserID(userID string) (*models.User, error) {
	var user models.User
	if err := database.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *UserDAO) GetByIdentifier(identifier string) (*models.User, error) {
	var user models.User
	if err := database.DB.Where("username = ? OR email = ?", identifier, identifier).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *UserDAO) UpdateByUserID(userID string, updates map[string]any) (int64, error) {
	result := database.DB.Model(&models.User{}).Where("user_id = ?", userID).Updates(updates)
	return result.RowsAffected, result.Error
}

func (d *UserDAO) DeleteByUserID(userID string) (int64, error) {
	result := database.DB.Where("user_id = ?", userID).Delete(&models.User{})
	return result.RowsAffected, result.Error
}

func (d *UserDAO) List(page, pageSize int) ([]models.User, int64, error) {
	_, pageSize, offset := normalizePagination(page, pageSize, 10)

	query := database.DB.Model(&models.User{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []models.User
	if err := query.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}
