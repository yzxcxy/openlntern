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
		&models.Agent{},
		&models.AgentBinding{},
		&models.MemorySyncState{},
		&models.MemoryUsageLog{},
		&models.SkillFrontmatter{},
		&models.Plugin{},
		&models.Tool{},
		&models.PluginDefault{},
		&models.ModelProvider{},
		&models.ModelCatalog{},
		&models.DefaultModelConfig{},
		&models.ThreadContextSnapshot{},
		&models.SandboxInstance{},
		&models.UserRuntimeConfig{},
	); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	// AutoMigrate 在不同 MySQL 版本上对 TEXT -> LONGTEXT 的放大并不稳定，这里显式修正消息表字段类型。
	if err := DB.Migrator().AlterColumn(&models.Message{}, "Content"); err != nil {
		return fmt.Errorf("alter message.content: %w", err)
	}
	if err := DB.Migrator().AlterColumn(&models.Message{}, "Metadata"); err != nil {
		return fmt.Errorf("alter message.metadata: %w", err)
	}
	return err
}
