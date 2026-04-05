package main

import (
	"context"
	"log"
	"openIntern/internal/config"
	"openIntern/internal/database"
	"openIntern/internal/routers"
	accountsvc "openIntern/internal/services/account"
	agentsvc "openIntern/internal/services/agent"
	memorysvc "openIntern/internal/services/memory"
	pluginsvc "openIntern/internal/services/plugin"
	storagesvc "openIntern/internal/services/storage"
)

func main() {
	cfg := config.LoadConfig("config.yaml")

	// 初始化运行时配置
	config.InitRuntime(cfg, "config.yaml")

	if err := database.Init(cfg.MySQL.DSN); err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	if err := database.InitRedis(cfg.Redis); err != nil {
		log.Fatalf("failed to init redis: %v", err)
	}
	accountsvc.InitAuth(cfg.JWT.Secret, cfg.JWT.ExpireMinutes)
	if err := storagesvc.InitFile(cfg.COS); err != nil {
		log.Fatalf("failed to init file service: %v", err)
	}
	pluginsvc.InitPlugin(cfg.Plugin)
	database.InitContextStore(cfg.Tools.OpenViking)
	memorysvc.InitMemory(cfg.Tools)
	memorysvc.InitMemorySync()
	shutdown, err := agentsvc.InitEino(cfg.SummaryLLM, cfg.Tools, cfg.Agent, cfg.ContextCompression, cfg.APMPlus)
	if err != nil {
		log.Fatalf("failed to init eino: %v", err)
	}
	if shutdown != nil {
		defer func() {
			_ = shutdown(context.Background())
		}()
	}

	r := routers.SetupRouter()
	r.Run(cfg.Port)
}
