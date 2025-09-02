package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/app"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/config"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/infra/db"
	"github.com/NastyaGoryachaya/crypto-rate-service/pkg/logger"
)

func main() {
	// 1) config
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "config/config.yaml"
	}

	cfg, err := config.LoadConfig(path)
	if err != nil {
		slog.Error("failed to read config", slog.String("path", path), slog.Any("err", err))
		os.Exit(1)
	}

	// 2) logger
	appLog := logger.New(&cfg.Logger)
	appLog.Info("starting crypto-rate-service")

	// 3) context + signals
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 4) db
	pool, err := db.NewPool(&cfg.Postgres)
	if err != nil {
		appLog.Error("db connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// 5) build application
	application, err := app.NewApp(*cfg, appLog, pool)
	if err != nil {
		appLog.Error("app init failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 6) run application
	if err := application.Run(ctx); err != nil {
		appLog.Error("application stopped with error", slog.String("error", err.Error()))
	}

	appLog.Info("crypto-rate-service stopped")
}
