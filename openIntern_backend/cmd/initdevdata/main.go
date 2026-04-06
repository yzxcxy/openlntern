package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"

	"openIntern/internal/config"
	"openIntern/internal/dao"
	"openIntern/internal/database"
	"openIntern/internal/models"
	accountsvc "openIntern/internal/services/account"
	pluginsvc "openIntern/internal/services/plugin"

	"gorm.io/gorm/logger"
)

func main() {
	configPath := flag.String("config", "config.yaml", "backend config file path")
	username := flag.String("username", "admin", "default username")
	email := flag.String("email", "admin@example.com", "default email")
	password := flag.String("password", "admin123456", "default password")
	phone := flag.String("phone", "", "default phone")
	flag.Parse()

	if err := run(*configPath, *username, *email, *password, *phone); err != nil {
		log.Fatalf("init dev data failed: %v", err)
	}
}

func run(configPath string, username string, email string, password string, phone string) error {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	password = strings.TrimSpace(password)
	phone = strings.TrimSpace(phone)

	// 初始化默认账号前先做必要的入参校验，避免写入半成品数据。
	if username == "" || email == "" || password == "" {
		return errors.New("username, email, and password are required")
	}

	cfg, err := config.LoadConfigStrict(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := database.Init(cfg.MySQL.DSN); err != nil {
		return fmt.Errorf("init database: %w", err)
	}
	// 初始化脚本只关心最终结果，屏蔽预期内的 not found SQL 日志，避免输出噪音。
	database.DB.Logger = database.DB.Logger.LogMode(logger.Silent)

	// 默认账号创建后会自动为用户挂载内建插件，因此这里需要先初始化插件服务。
	pluginsvc.InitPlugin(cfg.Plugin)

	existing, err := findExistingUser(username, email)
	if err != nil {
		return err
	}
	if existing != nil {
		fmt.Printf("default user already exists: username=%s email=%s user_id=%s\n", existing.Username, existing.Email, existing.UserID)
		return nil
	}

	user := &models.User{
		Username: username,
		Email:    email,
		Password: password,
		Phone:    phone,
	}
	if err := accountsvc.User.CreateUser(user); err != nil {
		return fmt.Errorf("create default user: %w", err)
	}

	fmt.Printf("default user created: username=%s email=%s user_id=%s\n", user.Username, user.Email, user.UserID)
	return nil
}

func findExistingUser(username string, email string) (*models.User, error) {
	// 优先按用户名查，再按邮箱查，保证脚本可以重复执行而不是因为唯一键直接失败。
	if username != "" {
		user, err := dao.User.GetByIdentifier(username)
		if err == nil {
			return user, nil
		}
	}
	if email != "" {
		user, err := dao.User.GetByIdentifier(email)
		if err == nil {
			return user, nil
		}
	}
	return nil, nil
}
