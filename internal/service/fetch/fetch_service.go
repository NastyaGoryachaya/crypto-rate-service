package fetch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"log/slog"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/domain"
)

type Service interface {
	FetchAndSaveCurrency(ctx context.Context) error
}

type RatesProvider interface {
	FetchRates(ctx context.Context) ([]RateQuote, error)
}

type CoinReader interface {
	GetAllCoins(ctx context.Context) ([]domain.Coin, error)
}

type PriceWriter interface {
	SavePrice(ctx context.Context, price domain.Price) error
}

type RateQuote struct {
	CoinSymbol string
	Value      float64
	Timestamp  time.Time
}

type fetchService struct {
	ratesProvider RatesProvider
	coinRepo      CoinReader
	priceRepo     PriceWriter
	logger        *slog.Logger
}

// NewService — конструктор сервиса для получения и сохранения курсов валют.
func NewService(ratesProvider RatesProvider, coinRepo CoinReader, priceRepo PriceWriter, logger *slog.Logger) Service {
	return &fetchService{
		ratesProvider: ratesProvider,
		coinRepo:      coinRepo,
		priceRepo:     priceRepo,
		logger:        logger,
	}
}

// FetchAndSaveCurrency — получает список монет, запрашивает их курсы у провайдера и сохраняет цены в БД.
func (s *fetchService) FetchAndSaveCurrency(ctx context.Context) error {
	coins, err := s.coinRepo.GetAllCoins(ctx)
	if err != nil {
		s.logger.Error("fetch all coins", "err", err)
		return fmt.Errorf("fetch all coins: %w", err)
	}

	rates, err := s.ratesProvider.FetchRates(ctx)
	if err != nil {
		s.logger.Error("fetch rates", "err", err)
		return fmt.Errorf("fetch rates: %w", err)
	}

	// создаём map для быстрого поиска цены по символу
	rateMap := make(map[string]RateQuote)
	for _, r := range rates {
		rateMap[strings.ToUpper(r.CoinSymbol)] = r
	}

	for _, coin := range coins {
		sym := strings.ToUpper(coin.Symbol)
		r, ok := rateMap[sym]
		if !ok {
			s.logger.Warn("missing rate for coin", "symbol", coin.Symbol)
			continue // нет данных по этой монете в ответе API
		}

		// предпочтительно API время; fallback to now UTC
		ts := r.Timestamp
		if ts.IsZero() {
			ts = time.Now().UTC()
		}

		price := domain.Price{
			CoinSymbol: coin.Symbol,
			Value:      r.Value,
			Timestamp:  ts,
		}

		if err := s.priceRepo.SavePrice(ctx, price); err != nil {
			s.logger.Warn("save price to db failed", "symbol", coin.Symbol, "err", err)
		}
	}
	return nil
}
