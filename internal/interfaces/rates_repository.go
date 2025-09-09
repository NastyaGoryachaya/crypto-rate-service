package interfaces

import (
	"context"
	"time"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/domain"
)

type Storage interface {
	// SaveCoins - Идемпотентное сохранение пачки снимков
	SaveCoins(ctx context.Context, items []domain.Coin) error

	// GetAllCoins - Актуальные цены для списка символов (последний снапшот на каждый символ)
	GetAllCoins(ctx context.Context) ([]domain.Coin, error)

	// GetCoinBySymbol - Актуальная цена для одного символа
	GetCoinBySymbol(ctx context.Context, symbol string) (domain.Coin, error)

	// History - История за окно (нужна сервису, чтобы посчитать min/max и процент за день или любой window)
	History(ctx context.Context, symbol string, from, to time.Time) ([]domain.Coin, error)
}
