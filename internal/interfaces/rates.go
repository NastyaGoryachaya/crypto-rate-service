package interfaces

import (
	"context"
	"time"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/domain"
)

// CryptoProvider — внешний источник курсов (например, CoinGecko API).
type CryptoProvider interface {
	FetchRates(ctx context.Context) ([]domain.Coin, error)
}

// Ingestion — интерфейс для планировщика обновления курсов.
type Ingestion interface {
	FetchAndSaveCurrency(ctx context.Context) error
}

// Storage — репозиторий для сохранения и выборки курсов из БД.
type Storage interface {
	SaveCoins(ctx context.Context, items []domain.Coin) error
	GetAllCoins(ctx context.Context) ([]domain.Coin, error)
	GetCoinBySymbol(ctx context.Context, symbol string) (domain.Coin, error)
	History(ctx context.Context, symbol string, from, to time.Time) ([]domain.Coin, error)
}

// Service — сервисный интерфейс для получения актуальных цен и статистики.
type Service interface {
	GetLatest(ctx context.Context) ([]domain.Coin, error)
	GetLatestBySymbol(ctx context.Context, symbol string, from, to time.Time) (latest domain.Coin, min float64, max float64, pct float64, err error)
}
