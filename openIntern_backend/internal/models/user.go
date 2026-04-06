package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID     uint   `gorm:"primarykey" json:"-"`                         // 内部自增ID，不对外暴露
	UserID string `gorm:"uniqueIndex;not null;size:36" json:"user_id"` // 业务ID，对外暴露为 user_id

	Username string `gorm:"uniqueIndex;not null;size:50" json:"username"`
	Email    string `gorm:"uniqueIndex;not null;size:100" json:"email"`
	Password string `gorm:"not null" json:"-"`      // 存储哈希后的密码
	Avatar   string `gorm:"size:255" json:"avatar"` // 头像对象 key
	Phone    string `gorm:"size:50" json:"phone"`   // 联系方式

	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate GORM hook to set UserID if not present
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.UserID == "" {
		u.UserID = uuid.New().String()
	}
	return
}
