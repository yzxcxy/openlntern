package main

import (
	"agent_backend/internal/config"
	"agent_backend/internal/routers"
)

func main() {
	cfg := config.LoadConfig()
	r := routers.SetupRouter()
	r.Run(cfg.Port)
}
