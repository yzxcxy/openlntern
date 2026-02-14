package main

import (
	"openIntern/internal/config"
	"openIntern/internal/routers"
)

func main() {
	cfg := config.LoadConfig()
	r := routers.SetupRouter()
	r.Run(cfg.Port)
}
