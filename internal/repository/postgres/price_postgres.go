package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/domain"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PriceRepo — репозиторий для работы с таблицей цен (prices).
type PriceRepo struct {
	db *pgxpool.Pool
}

// NewPriceRepository - Создаёт новый репозиторий цен на основе пула соединений.
func NewPriceRepository(db *pgxpool.Pool) *PriceRepo {
	return &PriceRepo{db: db}
}

// SavePrice - Сохраняет новую цену в таблицу prices.
func (r *PriceRepo) SavePrice(ctx context.Context, price domain.Price) error {
	query := `
            INSERT INTO prices (coin_symbol, value, timestamp)
            VALUES ($1, $2, $3)
            `

	_, err := r.db.Exec(ctx, query, price.CoinSymbol, price.Value, price.Timestamp)
	return err
}

// GetLatestPrice - Получить последнюю цену по символу монеты.
func (r *PriceRepo) GetLatestPrice(ctx context.Context, coinSymbol string) (*domain.Price, error) {
	query := `
        SELECT coin_symbol, value, timestamp
        FROM prices
        WHERE coin_symbol = $1
        ORDER BY timestamp DESC
        LIMIT 1
    `
	row := r.db.QueryRow(ctx, query, coinSymbol)

	var price domain.Price
	err := row.Scan(&price.CoinSymbol, &price.Value, &price.Timestamp)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &price, nil
}

// GetMinAndMaxPrices - Получить минимальную и максимальную цену с указанного момента времени.
func (r *PriceRepo) GetMinAndMaxPrices(ctx context.Context, coinSymbol string, since time.Time) (min, max domain.Price, err error) {
	// min
	minQuery := `
        SELECT coin_symbol, value, timestamp
        FROM prices
        WHERE coin_symbol = $1 AND timestamp >= $2
        ORDER BY value
        LIMIT 1
    `
	row := r.db.QueryRow(ctx, minQuery, coinSymbol, since)
	err = row.Scan(&min.CoinSymbol, &min.Value, &min.Timestamp)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Price{}, domain.Price{}, repository.ErrNotFound
	}
	if err != nil {
		return domain.Price{}, domain.Price{}, err
	}

	// max
	maxQuery := `
        SELECT coin_symbol, value, timestamp
        FROM prices
        WHERE coin_symbol = $1 AND timestamp >= $2
        ORDER BY value DESC
        LIMIT 1
    `
	row = r.db.QueryRow(ctx, maxQuery, coinSymbol, since)
	err = row.Scan(&max.CoinSymbol, &max.Value, &max.Timestamp)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Price{}, domain.Price{}, repository.ErrNotFound
	}
	if err != nil {
		return domain.Price{}, domain.Price{}, err
	}
	return min, max, nil
}

// GetPriceBefore - Получить цену по символу до указанного времени (ближайшая к этому моменту).
func (r *PriceRepo) GetPriceBefore(ctx context.Context, coinSymbol string, before time.Time) (*domain.Price, error) {
	query := `
        SELECT coin_symbol, value, timestamp
        FROM prices
        WHERE coin_symbol = $1 AND timestamp <= $2
        ORDER BY timestamp DESC
        LIMIT 1
    `

	row := r.db.QueryRow(ctx, query, coinSymbol, before)

	var price domain.Price
	err := row.Scan(&price.CoinSymbol, &price.Value, &price.Timestamp)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &price, nil
}
