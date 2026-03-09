package database

import (
	"fmt"
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
		return fmt.Errorf("connect database: %w", err)
	}

	// 自动迁移表结构
	if err := DB.AutoMigrate(
		&models.User{},
		&models.A2UI{},
		&models.Thread{},
		&models.Message{},
		&models.SkillFrontmatter{},
		&models.Plugin{},
		&models.Tool{},
		&models.PluginDefault{},
		&models.ModelProvider{},
		&models.ModelCatalog{},
		&models.DefaultModelConfig{},
		&models.ThreadContextSnapshot{},
	); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	if DB.Migrator().HasColumn(&models.A2UI{}, "type") {
		if err := DB.Migrator().DropColumn(&models.A2UI{}, "type"); err != nil {
			return fmt.Errorf("drop a2ui.type: %w", err)
		}
	}
	if DB.Migrator().HasColumn(&models.A2UI{}, "user_id") {
		if err := DB.Migrator().DropColumn(&models.A2UI{}, "user_id"); err != nil {
			return fmt.Errorf("drop a2ui.user_id: %w", err)
		}
	}
	if DB.Migrator().HasColumn(&models.Thread{}, "owner_id") {
		if err := DB.Migrator().DropColumn(&models.Thread{}, "owner_id"); err != nil {
			return fmt.Errorf("drop thread.owner_id: %w", err)
		}
	}

	return err
}
