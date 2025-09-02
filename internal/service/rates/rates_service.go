package rates

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/domain"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/repository"
)

// Бизнес-логика - получение курсов, расчёт изменений

type Service interface {
	// GetAllRates — отдаёт последние курсы по всем монетам
	GetAllRates(ctx context.Context) ([]RateStats, error)
	// GetRateBySymbol — текущая цена, мин/макс за 24ч и изменение за 1ч по символу
	GetRateBySymbol(ctx context.Context, symbol string) (RateStats, error)
}

type CoinReader interface {
	GetAllCoins(ctx context.Context) ([]domain.Coin, error)
	GetCoinBySymbol(ctx context.Context, symbol string) (*domain.Coin, error)
}

type PriceReader interface {
	GetLatestPrice(ctx context.Context, coinSymbol string) (*domain.Price, error)
	GetMinAndMaxPrices(ctx context.Context, coinSymbol string, since time.Time) (min, max domain.Price, err error)
	GetPriceBefore(ctx context.Context, coinSymbol string, before time.Time) (*domain.Price, error)
}

type RateStats struct {
	Symbol      string
	Price       float64
	Min24h      float64
	Max24h      float64
	Change1hPct float64
	UpdatedAt   time.Time
}

type service struct {
	coinRepo  CoinReader
	priceRepo PriceReader
	clock     Clock
	logger    *slog.Logger
}

func NewService(coinRepo CoinReader, priceRepo PriceReader, logger *slog.Logger) Service {
	return &service{
		coinRepo:  coinRepo,
		priceRepo: priceRepo,
		clock:     NewRealClock(),
		logger:    logger,
	}
}

// NewServiceWithClock - Конструктор для тестов: позволяет подставить фиксированные "часы".
func NewServiceWithClock(coinRepo CoinReader, priceRepo PriceReader, clk Clock, logger *slog.Logger) Service {
	return &service{
		coinRepo:  coinRepo,
		priceRepo: priceRepo,
		clock:     clk,
		logger:    logger,
	}
}

func (s *service) GetAllRates(ctx context.Context) ([]RateStats, error) {
	coins, err := s.coinRepo.GetAllCoins(ctx)
	if err != nil {
		s.logger.Error("failed to get all coins", "err", err)
		return nil, err
	}
	if len(coins) == 0 {
		s.logger.Warn("no coins configured")
		return []RateStats{}, nil
	}

	out := make([]RateStats, 0, len(coins))
	for _, coin := range coins {
		price, err := s.priceRepo.GetLatestPrice(ctx, coin.Symbol)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				s.logger.Debug("latest price not found", "symbol", coin.Symbol)
				continue
			}
			s.logger.Error("failed to get latest price", "symbol", coin.Symbol, "err", err)
			continue
		}
		if price == nil {
			s.logger.Debug("latest price is nil", "symbol", coin.Symbol)
			continue
		}
		out = append(out, RateStats{
			Symbol:    coin.Symbol,
			Price:     price.Value,
			UpdatedAt: price.Timestamp,
		})
	}

	if len(out) == 0 {
		s.logger.Warn("no latest prices available")
		return nil, ErrPriceNotFound
	}
	s.logger.Info("computed stats for all coins", "count", len(out))
	return out, nil
}

func (s *service) GetRateBySymbol(ctx context.Context, symbol string) (RateStats, error) {
	// валидация/поиск монеты по символу (чтобы отдавать корректные ошибки)
	coin, err := s.coinRepo.GetCoinBySymbol(ctx, symbol)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.logger.Warn("coin not found", "symbol", symbol)
			return RateStats{}, ErrCoinNotFound
		}
		s.logger.Error("failed to get coin by symbol", "symbol", symbol, "err", err)
		return RateStats{}, err
	}
	if coin == nil {
		s.logger.Warn("coin not found (nil result)", "symbol", symbol)
		return RateStats{}, ErrCoinNotFound
	}

	now := s.clock.Now()

	// Мин/макс за последние 24 часа
	since24h := now.Add(-24 * time.Hour)
	minPrice, maxPrice, err := s.priceRepo.GetMinAndMaxPrices(ctx, coin.Symbol, since24h)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.logger.Debug("no min/max within 24h", "symbol", coin.Symbol, "since", since24h)
			return RateStats{}, ErrMinMaxPrice
		}
		s.logger.Error("failed to get min and max prices", "symbol", coin.Symbol, "since", since24h, "err", err)
		return RateStats{}, ErrMinMaxPrice
	}

	// Цена час назад
	oldBefore := now.Add(-1 * time.Hour)
	oldPrice, err := s.priceRepo.GetPriceBefore(ctx, coin.Symbol, oldBefore)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.logger.Debug("no price an hour ago", "symbol", coin.Symbol, "at", oldBefore)
			return RateStats{}, ErrPriceNotFound
		}
		s.logger.Error("failed to get price an hour ago", "symbol", coin.Symbol, "at", oldBefore, "err", err)
		return RateStats{}, err
	}
	if oldPrice == nil {
		s.logger.Debug("price an hour ago is nil", "symbol", coin.Symbol, "at", oldBefore)
		return RateStats{}, ErrPriceNotFound
	}
	if oldPrice.Value == 0 {
		s.logger.Warn("old price is zero, cannot compute percentage change", "symbol", coin.Symbol)
		return RateStats{}, ErrHourAgoPrice
	}

	// Текущая цена
	currentPrice, err := s.priceRepo.GetLatestPrice(ctx, coin.Symbol)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.logger.Debug("no current price", "symbol", coin.Symbol)
			return RateStats{}, ErrPriceNotFound
		}
		s.logger.Error("failed to get current price", "symbol", coin.Symbol, "err", err)
		return RateStats{}, err
	}
	if currentPrice == nil {
		s.logger.Debug("current price is nil", "symbol", coin.Symbol)
		return RateStats{}, ErrPriceNotFound
	}

	changePct := ((currentPrice.Value - oldPrice.Value) / oldPrice.Value) * 100

	s.logger.Info("computed stats",
		"symbol", coin.Symbol,
		"min", minPrice.Value,
		"max", maxPrice.Value,
		"change_pct", changePct,
	)

	return RateStats{
		Symbol:      coin.Symbol,
		Price:       currentPrice.Value,
		Min24h:      minPrice.Value,
		Max24h:      maxPrice.Value,
		Change1hPct: changePct,
		UpdatedAt:   currentPrice.Timestamp,
	}, nil
}
