package interfaces

import (
	"context"
	"time"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/domain"
)

type Service interface {
	// GetLatest - Актуальные цены для списка
	GetLatest(ctx context.Context) ([]domain.Coin, error)

	// GetLatestBySymbol - Актуальная цена для одного: symbol, price, min/max, pct, time
	GetLatestBySymbol(ctx context.Context, symbol string, from, to time.Time) (latest domain.Coin, min float64, max float64, pct float64, err error)
}
