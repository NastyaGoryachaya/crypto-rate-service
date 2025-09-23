package rates

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/domain"
	errs "github.com/NastyaGoryachaya/crypto-rate-service/internal/errors"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/interfaces"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/pkg/utils"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	storage        interfaces.Storage
	cryptoProvider interfaces.CryptoProvider
	logger         *slog.Logger
}

func NewService(storage interfaces.Storage, provider interfaces.CryptoProvider, logger *slog.Logger) *Service {
	return &Service{
		storage:        storage,
		cryptoProvider: provider,
		logger:         logger,
	}
}

// FetchAndSaveCurrency — получает список монет, запрашивает их курсы у провайдера и сохраняет цены в БД.
func (s *Service) FetchAndSaveCurrency(ctx context.Context) error {
	symbols, err := s.storage.GetAllCoins(ctx)
	if err != nil {
		s.logger.Error("failed to get all coins from storage", "err", err)
		return fmt.Errorf("%w: storage.GetAllCoins: %w", errs.ErrInternal, err)
	}

	rates, err := s.cryptoProvider.FetchRates(ctx)
	if err != nil {
		s.logger.Error("fetch rates", "err", err)
		return fmt.Errorf("%w: provider.FetchRates: %w", errs.ErrInternal, err)
	}

	// создаём map для быстрого поиска цены по символу
	rateMap := make(map[string]domain.Coin)
	for _, r := range rates {
		rateMap[strings.ToUpper(r.Symbol)] = r
	}

	items := make([]domain.Coin, 0, len(symbols))
	now := utils.NowFunc()
	for _, sym := range symbols {
		u := strings.ToUpper(sym.Symbol)
		r, ok := rateMap[u]
		if !ok {
			s.logger.Warn("missing rate for coin", "symbol", u)
			continue
		}
		r.Symbol = u
		if r.UpdatedAt.IsZero() {
			r.UpdatedAt = now
		}
		items = append(items, r)
	}

	if err := s.storage.SaveCoins(ctx, items); err != nil {
		s.logger.Error("save prices to db failed", "count", len(items), "err", err)
		return fmt.Errorf("%w: storage.SaveCoins(count=%d): %w", errs.ErrInternal, len(items), err)
	}
	s.logger.Info("rates saved", "count", len(items))

	return nil
}

func (s *Service) GetLatest(ctx context.Context) ([]domain.Coin, error) {
	items, err := s.storage.GetAllCoins(ctx)
	if err != nil {
		s.logger.Error("failed to get all coins", "err", err)
		return nil, fmt.Errorf("%w: storage.GetAllCoins: %w", errs.ErrInternal, err)
	}
	if len(items) == 0 {
		s.logger.Warn("no latest coins available")
		return nil, errs.ErrPriceNotFound
	}
	s.logger.Info("loaded latest coins", "count", len(items))
	return items, nil
}

func (s *Service) GetLatestBySymbol(ctx context.Context, symbol string, from, to time.Time) (latest domain.Coin, min float64, max float64, pct float64, err error) {
	// Нормализуем символ
	symbol = strings.ToUpper(symbol)

	// Нормализуем окно времени: если не задано — последние 24 часа; всегда UTC
	if to.IsZero() {
		to = utils.NowFunc()
	} else {
		to = to.UTC()
	}
	if from.IsZero() {
		from = to.Add(-24 * time.Hour)
	} else {
		from = from.UTC()
	}

	// Текущая цена на момент `to`
	latest, err = s.storage.GetCoinBySymbol(ctx, symbol)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logger.Warn("coin not found", "symbol", symbol)
			return domain.Coin{}, 0, 0, 0, errs.ErrCoinNotFound
		}
		s.logger.Error("failed to get coin by symbol", "symbol", symbol, "err", err)
		return domain.Coin{}, 0, 0, 0, fmt.Errorf("%w: storage.GetCoinBySymbol(%s): %w", errs.ErrInternal, symbol, err)
	}

	// История в окне [from..to]
	s.logger.Debug("loading history window", "symbol", symbol, "from", from, "to", to)
	rows, err := s.storage.History(ctx, symbol, from, to)
	if err != nil {
		return domain.Coin{}, 0, 0, 0, fmt.Errorf("%w: storage.History(%s): %w", errs.ErrInternal, symbol, err)
	}
	if len(rows) == 0 {
		return domain.Coin{}, 0, 0, 0, errs.ErrPriceNotFound
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

	// Процент за последний час на момент `to` (ищем самую позднюю точку <= threshold)
	threshold := to.Add(-1 * time.Hour)
	var (
		prevPrice float64
		prevFound bool
		prevAt    time.Time
	)
	for i := 0; i < len(rows); i++ {
		ts := rows[i].UpdatedAt
		if !ts.After(threshold) {
			if !prevFound || ts.After(prevAt) || ts.Equal(prevAt) {
				prevPrice = rows[i].Price
				prevAt = ts
				prevFound = true
			}
		}
	}
	if !prevFound || prevPrice == 0 {
		s.logger.Warn("insufficient data for pct", "symbol", symbol, "threshold", threshold)
		// Не роняем ответ: возвращаем latest/min/max, а pct=0 (транспорт может отдать null)
		pct = 0
	} else {
		pct = ((latest.Price - prevPrice) / prevPrice) * 100
	}

	s.logger.Info("computed stats", "symbol", symbol, "min", min, "max", max, "pct", pct)
	return latest, min, max, pct, nil
}
