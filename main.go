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
	services.InitAuth(cfg.JWT.Secret, cfg.JWT.ExpireMinutes)
	r := routers.SetupRouter()
	r.Run(cfg.Port)
}
