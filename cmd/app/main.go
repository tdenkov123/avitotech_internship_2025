package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/tdenkov123/avitotech_internship_2025/internal/config"
	httpserver "github.com/tdenkov123/avitotech_internship_2025/internal/http_server"
	"github.com/tdenkov123/avitotech_internship_2025/internal/logger"
	"go.uber.org/zap"
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

	srv := httpserver.New(cfg, logg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logg.Info("starting HTTP server", zap.String("port", cfg.ServerPort))

	if err := srv.Run(ctx); err != nil {
		logg.Fatal("server stopped with error", zap.Error(err))
	}

	logg.Info("server stopped")
}
