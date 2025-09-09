package interfaces

import (
	"context"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/domain"
)

type CryptoProvider interface {
	FetchRates(ctx context.Context) ([]domain.Coin, error)
}

type Ingestion interface {
	FetchAndSaveCurrency(ctx context.Context) error
}
