package postgres

import (
	"context"
	"time"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CoinRepo struct {
	db *pgxpool.Pool
}

func NewCoinRepo(db *pgxpool.Pool) *CoinRepo {
	return &CoinRepo{db: db}
}

// SaveCoins — сохранить список курсов в таблицу prices.
func (r *CoinRepo) SaveCoins(ctx context.Context, items []domain.Coin) error {
	if len(items) == 0 {
		return nil
	}

	const query = `
		INSERT INTO prices (coin_symbol, value, timestamp)
		VALUES ($1, $2, $3)
		ON CONFLICT (coin_symbol, timestamp)
		DO UPDATE SET value = EXCLUDED.value
	`

	for _, it := range items {
		if _, err := r.db.Exec(ctx, query, it.Symbol, it.Price, it.UpdatedAt); err != nil {
			return err
		}
	}
	return nil
}

// GetAllCoins — получить последние цены для всех монет.
func (r *CoinRepo) GetAllCoins(ctx context.Context) ([]domain.Coin, error) {
	const query = `
		SELECT DISTINCT ON (coin_symbol)
		       coin_symbol, value, timestamp
		FROM prices
		ORDER BY coin_symbol, timestamp DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Coin
	for rows.Next() {
		var c domain.Coin
		if err := rows.Scan(&c.Symbol, &c.Price, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return out, nil
}

// GetCoinBySymbol — получить последнюю цену по символу монеты.
func (r *CoinRepo) GetCoinBySymbol(ctx context.Context, symbol string) (domain.Coin, error) {
	const query = `
		SELECT coin_symbol, value, timestamp
		FROM prices
		WHERE coin_symbol = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`
	var c domain.Coin
	err := r.db.QueryRow(ctx, query, symbol).Scan(&c.Symbol, &c.Price, &c.UpdatedAt)
	return c, err
}

// History — получить историю цен по монете за указанный период.
func (r *CoinRepo) History(ctx context.Context, symbol string, from, to time.Time) ([]domain.Coin, error) {
	const query = `
		SELECT coin_symbol, value, timestamp
		FROM prices
		WHERE coin_symbol = $1
		  AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp
	`
	rows, err := r.db.Query(ctx, query, symbol, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Coin
	for rows.Next() {
		var c domain.Coin
		if err := rows.Scan(&c.Symbol, &c.Price, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return out, nil
}
