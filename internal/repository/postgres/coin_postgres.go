package postgres

import (
	"context"
	"errors"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/domain"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CoinRepo struct {
	db *pgxpool.Pool
}

// NewCoinRepository - Создаёт новый репозиторий монет на основе пула соединений.
func NewCoinRepository(db *pgxpool.Pool) *CoinRepo {
	return &CoinRepo{db: db}
}

// GetAllCoins - Получить список всех монет из таблицы coins
func (r *CoinRepo) GetAllCoins(ctx context.Context) ([]domain.Coin, error) {
	query := `SELECT symbol FROM coins ORDER BY symbol;`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var coins []domain.Coin
	for rows.Next() {
		var coin domain.Coin
		if err := rows.Scan(&coin.Symbol); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, repository.ErrNotFound
			}
			return nil, err
		}
		coins = append(coins, coin)
	}
	return coins, nil
}

// GetCoinBySymbol - Найти монету по символу
func (r *CoinRepo) GetCoinBySymbol(ctx context.Context, symbol string) (*domain.Coin, error) {
	query := `SELECT symbol FROM coins WHERE UPPER(symbol) = UPPER($1);`
	row := r.db.QueryRow(ctx, query, symbol)

	var coin domain.Coin
	if err := row.Scan(&coin.Symbol); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &coin, nil
}
