package main

import (
	"log"
	"openIntern/internal/config"
	"openIntern/internal/database"
	"openIntern/internal/routers"
	"openIntern/internal/services"
)

func main() {
	cfg := config.LoadConfig("config.yaml")
	if err := database.Init(cfg.MySQL.DSN); err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	if err := database.InitRedis(cfg.Redis); err != nil {
		log.Fatalf("failed to init redis: %v", err)
	}
	services.InitAuth(cfg.JWT.Secret, cfg.JWT.ExpireMinutes)
	if err := services.InitFile(cfg.COS); err != nil {
		log.Fatalf("failed to init file service: %v", err)
	}
	if err := services.InitEino(cfg.LLM); err != nil {
		log.Fatalf("failed to init eino: %v", err)
	}
	r := routers.SetupRouter()
	r.Run(cfg.Port)
}
