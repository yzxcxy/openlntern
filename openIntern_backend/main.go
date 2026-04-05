package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"openIntern/internal/config"
	"openIntern/internal/database"
	"openIntern/internal/routers"
	accountsvc "openIntern/internal/services/account"
	agentsvc "openIntern/internal/services/agent"
	memorysvc "openIntern/internal/services/memory"
	pluginsvc "openIntern/internal/services/plugin"
	openvikingsvc "openIntern/internal/services/openviking"
	storagesvc "openIntern/internal/services/storage"
	"strings"
	"syscall"
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

	// 初始化并启动 OpenViking 服务
	ovConfig := config.GetOpenVikingServiceConfig()
	healthURL := strings.TrimRight(strings.TrimSpace(cfg.Tools.OpenViking.BaseURL), "/") + "/health"
	ovManager := openvikingsvc.InitManager(ovConfig, config.GetOpenVikingConfigPath(), healthURL)
	if ovManager.ManagedExternally() {
		log.Println("openviking is managed externally; skip local process startup")
	} else {
		if err := ovManager.Start(); err != nil {
			log.Printf("warning: failed to start openviking: %v", err)
		}
	}

	// 设置信号处理，确保子进程被正确清理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		if !ovManager.ManagedExternally() {
			log.Println("received shutdown signal, stopping openviking...")
			if err := ovManager.Stop(); err != nil {
				log.Printf("warning: failed to stop openviking: %v", err)
			}
		}
		os.Exit(0)
	}()

	r := routers.SetupRouter()
	r.Run(cfg.Port)
}
