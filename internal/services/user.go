package services

import (
	"errors"
	"openIntern/internal/database"
	"openIntern/internal/models"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct{}

var User = new(UserService)

// CreateUser 创建用户
func (s *UserService) CreateUser(user *models.User) error {
	// 检查用户名或邮箱是否已存在
	var count int64
	database.DB.Model(&models.User{}).Where("username = ? OR email = ?", user.Username, user.Email).Count(&count)
	if count > 0 {
		return errors.New("username or email already exists")
	}

	// 密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)

	return database.DB.Create(user).Error
}

// GetUserByUserID 根据 UserID 获取用户
func (s *UserService) GetUserByUserID(userID string) (*models.User, error) {
	var user models.User
	err := database.DB.Where("user_id = ?", userID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) Authenticate(identifier, password string) (*models.User, error) {
	if identifier == "" || password == "" {
		return nil, errors.New("identifier and password are required")
	}
	var user models.User
	err := database.DB.Where("username = ? OR email = ?", identifier, identifier).First(&user).Error
	if err != nil {
		return nil, errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}
	return &user, nil
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(userID string, updates map[string]interface{}) error {
	// 如果包含密码，需要重新哈希
	if password, ok := updates["password"].(string); ok && password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		updates["password"] = string(hashedPassword)
	}

	result := database.DB.Model(&models.User{}).Where("user_id = ?", userID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}
	return nil
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(userID string) error {
	result := database.DB.Where("user_id = ?", userID).Delete(&models.User{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}
	return nil
}

// ListUsers 获取用户列表（分页）
func (s *UserService) ListUsers(page, pageSize int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	offset := (page - 1) * pageSize

	db := database.DB.Model(&models.User{})

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := db.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}
