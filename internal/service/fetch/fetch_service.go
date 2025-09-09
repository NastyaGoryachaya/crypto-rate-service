package fetch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"log/slog"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/domain"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/interfaces"
)

type Service struct {
	cryptoProvider interfaces.CryptoProvider
	storage        interfaces.Storage
	logger         *slog.Logger
}

// NewService — конструктор сервиса для получения и сохранения курсов валют.
func NewService(cryptoProvider interfaces.CryptoProvider, storage interfaces.Storage, logger *slog.Logger) *Service {
	return &Service{
		cryptoProvider: cryptoProvider,
		storage:        storage,
		logger:         logger,
	}
}

// FetchAndSaveCurrency — получает список монет, запрашивает их курсы у провайдера и сохраняет цены в БД.
func (s *Service) FetchAndSaveCurrency(ctx context.Context) error {
	coins, err := s.storage.GetAllCoins(ctx)
	if err != nil {
		s.logger.Error("fetch all coins", "err", err)
		return fmt.Errorf("fetch all coins: %w", err)
	}

	rates, err := s.cryptoProvider.FetchRates(ctx)
	if err != nil {
		s.logger.Error("fetch rates", "err", err)
		return fmt.Errorf("fetch rates: %w", err)
	}

	// создаём map для быстрого поиска цены по символу
	rateMap := make(map[string]domain.Coin)
	for _, r := range rates {
		rateMap[strings.ToUpper(r.Symbol)] = r
	}

	items := make([]domain.Coin, 0, len(coins))

	for _, coin := range coins {
		sym := strings.ToUpper(coin.Symbol)
		r, ok := rateMap[sym]
		if !ok {
			s.logger.Warn("missing rate for coin", "symbol", coin.Symbol)
			continue // нет данных по этой монете в ответе API
		}

		price := r
		price.Symbol = coin.Symbol
		if price.UpdatedAt.IsZero() {
			price.UpdatedAt = time.Now().UTC()
		}

		items = append(items, price)
	}

	if err := s.storage.SaveCoins(ctx, items); err != nil {
		s.logger.Warn("save prices to db failed", "count", len(items), "err", err)
	}

	return nil
}
