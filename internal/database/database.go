package database

import (
	"log"
	"openIntern/internal/models"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var DB *gorm.DB

// Init 初始化数据库连接
// 使用 MySQL
func Init(dsn string) error {
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		// 禁用创建物理外键约束，仅保持代码层面的逻辑关联
		DisableForeignKeyConstraintWhenMigrating: true,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// 自动迁移表结构
	if err := DB.AutoMigrate(&models.User{}, &models.A2UI{}, &models.Thread{}, &models.Message{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	return err
}
