package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tdenkov123/avitotech_internship_2025/internal/config"
	httpserver "github.com/tdenkov123/avitotech_internship_2025/internal/http_server"
	"github.com/tdenkov123/avitotech_internship_2025/internal/logger"
	"github.com/tdenkov123/avitotech_internship_2025/internal/service"
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	svc := service.New(dbPool)
	srv := httpserver.New(cfg, logg, svc)

	logg.Info("starting HTTP server", zap.String("port", cfg.ServerPort))

	if err := srv.Run(ctx); err != nil {
		logg.Fatal("server stopped with error", zap.Error(err))
	}

	logg.Info("server stopped")
}
