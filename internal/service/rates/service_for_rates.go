package rates

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/domain"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/interfaces"
)

type Service struct {
	storage interfaces.Storage
	logger  *slog.Logger
}

func NewService(storage interfaces.Storage, logger *slog.Logger) *Service {
	return &Service{
		storage: storage,
		logger:  logger,
	}
}

func (s *Service) GetLatest(ctx context.Context) ([]domain.Coin, error) {
	items, err := s.storage.GetAllCoins(ctx)
	if err != nil {
		s.logger.Error("failed to get all coins", "err", err)
		return nil, err
	}
	if len(items) == 0 {
		s.logger.Warn("no latest coins available")
		return nil, domain.ErrPriceNotFound
	}
	s.logger.Info("loaded latest coins", "count", len(items))
	return items, nil
}

func (s *Service) GetLatestBySymbol(ctx context.Context, symbol string, from, to time.Time) (latest domain.Coin, min float64, max float64, pct float64, err error) {
	// Нормализуем символ
	symbol = strings.ToUpper(symbol)

	// Текущая цена на момент `to`
	latest, err = s.storage.GetCoinBySymbol(ctx, symbol)
	if err != nil {
		return domain.Coin{}, 0, 0, 0, err
	}

	// История в окне [from..to]
	rows, err := s.storage.History(ctx, symbol, from, to)
	if err != nil {
		return domain.Coin{}, 0, 0, 0, err
	}
	if len(rows) == 0 {
		return domain.Coin{}, 0, 0, 0, domain.ErrPriceNotFound
	}

	// Мин/Макс за окно
	min, max = rows[0].Price, rows[0].Price
	for i := 1; i < len(rows); i++ {
		p := rows[i].Price
		if p < min {
			min = p
		}
		if p > max {
			max = p
		}
	}

	// Процент за последний час на момент `to`
	threshold := to.Add(-1 * time.Hour)
	var prev *domain.Coin
	for i := len(rows) - 1; i >= 0; i-- {
		if rows[i].UpdatedAt.Before(threshold) || rows[i].UpdatedAt.Equal(threshold) {
			prev = &rows[i]
			break
		}
	}
	if prev == nil || prev.Price == 0 {
		return domain.Coin{}, 0, 0, 0, domain.ErrPriceNotFound
	}
	pct = ((latest.Price - prev.Price) / prev.Price) * 100

	s.logger.Info("computed stats", "symbol", symbol, "min", min, "max", max, "pct", pct)
	return latest, min, max, pct, nil
}
