package db

import (
	"context"
	"fmt"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Подключение к базе данных

func NewPool(cfg *config.PostgresConfig) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}
	return pool, nil
}
