package account

import (
	"errors"
	"openIntern/internal/dao"
	"openIntern/internal/models"
	pluginsvc "openIntern/internal/services/plugin"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct{}

var User = new(UserService)

// CreateUser 创建用户
func (s *UserService) CreateUser(user *models.User) error {
	exists, err := dao.User.ExistsByUsernameOrEmail(user.Username, user.Email)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("username or email already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)

	if err := dao.User.Create(user); err != nil {
		return err
	}
	if err := pluginsvc.Plugin.EnsureBuiltinPluginsForUser(user.UserID); err != nil {
		return err
	}
	return nil
}

// GetUserByUserID 根据 UserID 获取用户
func (s *UserService) GetUserByUserID(userID string) (*models.User, error) {
	return dao.User.GetByUserID(userID)
}

func (s *UserService) Authenticate(identifier, password string) (*models.User, error) {
	if identifier == "" || password == "" {
		return nil, errors.New("identifier and password are required")
	}
	user, err := dao.User.GetByIdentifier(identifier)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}
	return user, nil
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

	rowsAffected, err := dao.User.UpdateByUserID(userID, updates)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("user not found")
	}
	return nil
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(userID string) error {
	rowsAffected, err := dao.User.DeleteByUserID(userID)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("user not found")
	}
	return nil
}

// ListUsers 获取用户列表（分页）
func (s *UserService) ListUsers(page, pageSize int) ([]models.User, int64, error) {
	return dao.User.List(page, pageSize)
}
