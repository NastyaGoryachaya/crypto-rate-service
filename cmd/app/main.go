package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/app"
	"github.com/labstack/gommon/log"
)

func main() {

	// context + signals
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// build application
	application, err := app.NewApp()
	if err != nil {
		log.Error("app init failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// run application
	if err := application.Run(ctx); err != nil {
		log.Error("application stopped with error", slog.String("error", err.Error()))
	}

	log.Info("crypto-rate-service stopped")
}
