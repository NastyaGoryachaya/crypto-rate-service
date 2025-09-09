package fetch_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/domain"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/service/fetch"
	fetchmocks "github.com/NastyaGoryachaya/crypto-rate-service/internal/service/fetch/mocks"
	"github.com/golang/mock/gomock"
)

// Success: пришли курсы для всех запрошенных монет, обе цены сохранились
func TestFetchAndSaveCurrency_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	coinRepo := fetchmocks.NewMockCoinReader(ctrl)
	priceRepo := fetchmocks.NewMockPriceWriter(ctrl)
	api := fetchmocks.NewMockRatesProvider(ctrl)

	coins := []domain.Coin{{Symbol: "BTC"}, {Symbol: "ETH"}}
	rates := []domain.Price{
		{CoinSymbol: "BTC", Value: 70000},
		{CoinSymbol: "ETH", Value: 3500},
	}

	// Ожидание
	// Ожидаем, что сервис получит монеты и запросит курсы
	api.EXPECT().
		FetchRates(gomock.Any()).
		Return(rates, nil).
		Times(1)

	coinRepo.EXPECT().
		GetAllCoins(gomock.Any()).
		Return(coins, nil).
		Times(1)

	// Должны быть два сохранения (BTC и ETH). Проверим корректность аргументов
	priceRepo.EXPECT().
		SavePrice(gomock.Any(), gomock.AssignableToTypeOf(domain.Price{})).
		DoAndReturn(func(_ context.Context, p domain.Price) error {
			if p.CoinSymbol == "BTC" && p.Value != 70000 {
				t.Errorf("BTC value mismatch: %v", p.Value)
			}
			if p.CoinSymbol == "ETH" && p.Value != 3500 {
				t.Errorf("ETH value mismatch: %v", p.Value)
			}
			if p.Timestamp.IsZero() {
				t.Error("timestamp must be set")
			}
			return nil
		}).
		Times(2) // BTC и ETH

	svc := fetch.NewService(api, coinRepo, priceRepo, slog.Default())

	// Проверка: ошибок быть не должно
	if err := svc.FetchAndSaveCurrency(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// MissingRateForCoin: курс для одной монеты отсутствует
// Сохраняем, только те монеты, для которых курс пришёл (BTC)
func TestFetchAndSaveCurrency_MissingRateForCoin(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	coinRepo := fetchmocks.NewMockCoinReader(ctrl)
	priceRepo := fetchmocks.NewMockPriceWriter(ctrl)
	api := fetchmocks.NewMockRatesProvider(ctrl)

	coins := []domain.Coin{{Symbol: "BTC"}, {Symbol: "ETH"}}
	rates := []domain.Price{{CoinSymbol: "BTC", Value: 70000}} // ETH отсутствует

	api.EXPECT().FetchRates(gomock.Any()).Return(rates, nil).Times(1)
	coinRepo.EXPECT().GetAllCoins(gomock.Any()).Return(coins, nil).Times(1)

	// Должен сохраниться только BTC
	priceRepo.EXPECT().
		SavePrice(gomock.Any(), gomock.AssignableToTypeOf(domain.Price{})).
		DoAndReturn(func(_ context.Context, p domain.Price) error {
			if p.CoinSymbol != "BTC" {
				t.Errorf("unexpected coin saved: %+v", p.CoinSymbol)
			}
			return nil
		}).
		Times(1)

	svc := fetch.NewService(api, coinRepo, priceRepo, slog.Default())

	if err := svc.FetchAndSaveCurrency(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// CoinRepoError: падаем на шаге чтения монет из БД
// В этом случае к API идти нельзя, и сохранений быть не должно
func TestFetchAndSaveCurrency_CoinRepoError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	coinRepo := fetchmocks.NewMockCoinReader(ctrl)
	priceRepo := fetchmocks.NewMockPriceWriter(ctrl)
	api := fetchmocks.NewMockRatesProvider(ctrl)

	// Сразу ошибка из БД при чтении списка монет
	coinRepo.EXPECT().GetAllCoins(gomock.Any()).Return(nil, errors.New("db failure")).Times(1)

	// Так как монеты не прочитаны — к API не ходим и ничего не сохраняем
	api.EXPECT().FetchRates(gomock.Any()).Times(0)
	priceRepo.EXPECT().SavePrice(gomock.Any(), gomock.Any()).Times(0)

	svc := fetch.NewService(api, coinRepo, priceRepo, slog.Default())

	// ОЖИДАЕМ ошибку от сервиса
	if err := svc.FetchAndSaveCurrency(ctx); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// ApiError: монеты из БД получили, но внешний API упал
// В этом случае сохранений быть не должно, и сервис возвращает ошибку
func TestFetchAndSaveCurrency_ApiError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	coinRepo := fetchmocks.NewMockCoinReader(ctrl)
	priceRepo := fetchmocks.NewMockPriceWriter(ctrl)
	api := fetchmocks.NewMockRatesProvider(ctrl)

	// Из БД монеты пришли
	coinRepo.EXPECT().GetAllCoins(gomock.Any()).Return([]domain.Coin{{Symbol: "BTC"}}, nil).Times(1)

	// API падает
	api.EXPECT().FetchRates(gomock.Any()).Return(nil, errors.New("api timeout")).Times(1)

	// Сохранений быть не должно
	priceRepo.EXPECT().SavePrice(gomock.Any(), gomock.Any()).Times(0)

	svc := fetch.NewService(api, coinRepo, priceRepo, slog.Default())

	// ОЖИДАЕМ ошибку от сервиса
	if err := svc.FetchAndSaveCurrency(ctx); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// SaveErrorRerCoin: курсы пришли для всех монет,
// но одна запись в БД не сохранилась. Сервис не должен падать,
// а должен сохранить остальные и завершить без ошибки
func TestFetchAndSaveCurrency_SaveErrorRerCoin(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	coinRepo := fetchmocks.NewMockCoinReader(ctrl)
	priceRepo := fetchmocks.NewMockPriceWriter(ctrl)
	api := fetchmocks.NewMockRatesProvider(ctrl)

	coins := []domain.Coin{{Symbol: "BTC"}, {Symbol: "ETH"}}
	rates := []domain.Price{
		{CoinSymbol: "BTC", Value: 70000},
		{CoinSymbol: "ETH", Value: 3500},
	}

	// Обычные успешные ответы от БД и API
	api.EXPECT().FetchRates(gomock.Any()).Return(rates, nil).Times(1)
	coinRepo.EXPECT().GetAllCoins(gomock.Any()).Return(coins, nil).Times(1)

	// Симулируем ошибку при сохранении одной из монет.
	call := 0
	priceRepo.EXPECT().SavePrice(gomock.Any(), gomock.AssignableToTypeOf(domain.Price{})).
		DoAndReturn(func(_ context.Context, p domain.Price) error {
			call++
			// Второй вызов падает
			if call == 2 {
				return errors.New("insert failed")
			}
			return nil
		}).
		Times(2)

	svc := fetch.NewService(api, coinRepo, priceRepo, slog.Default())

	// По бизнес-логике частичная ошибка записи НЕ должна валить процесс
	if err := svc.FetchAndSaveCurrency(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// (Дополнительно) Проверка окна времени для timestamp — демонстрирует,
// что сервис проставляет время «сейчас». Это не основной сценарий,
// но помогает поймать регресс, если время станет задаваться неверно
func TestFetchAndSaveCurrency_TimestampWindow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	coinRepo := fetchmocks.NewMockCoinReader(ctrl)
	priceRepo := fetchmocks.NewMockPriceWriter(ctrl)
	api := fetchmocks.NewMockRatesProvider(ctrl)

	coins := []domain.Coin{{Symbol: "BTC"}}
	rates := []domain.Price{{CoinSymbol: "BTC", Value: 70000}}

	api.EXPECT().FetchRates(gomock.Any()).Return(rates, nil).Times(1)
	coinRepo.EXPECT().GetAllCoins(gomock.Any()).Return(coins, nil).Times(1)

	start := time.Now().UTC()
	priceRepo.EXPECT().SavePrice(gomock.Any(), gomock.AssignableToTypeOf(domain.Price{})).
		DoAndReturn(func(_ context.Context, p domain.Price) error {
			if p.Timestamp.Before(start) || time.Since(p.Timestamp) > 2*time.Second {
				t.Errorf("timestamp out of expected window: %v", p.Timestamp)
			}
			return nil
		}).
		Times(1)

	svc := fetch.NewService(api, coinRepo, priceRepo, slog.Default())

	if err := svc.FetchAndSaveCurrency(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
