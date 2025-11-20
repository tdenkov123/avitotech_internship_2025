package main

import (
	"log"

	"github.com/tdenkov123/avitotech_internship_2025/internal/config"
	"github.com/tdenkov123/avitotech_internship_2025/internal/logger"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logg, err := logger.New(cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer logg.Sync()
}
