package rates

import (
	"context"
	"errors"
	"testing"
	"time"

	"log/slog"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/domain"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/pkg/utils"
	ratesmocks "github.com/NastyaGoryachaya/crypto-rate-service/internal/service/rates/mocks"
	"github.com/golang/mock/gomock"
)

func setupSvc(t *testing.T, fixed time.Time) (context.Context, *gomock.Controller,
	*ratesmocks.MockCoinReader, *ratesmocks.MockPriceReader, Service) {
	t.Helper()
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	coinRepo := ratesmocks.NewMockCoinReader(ctrl)
	priceRepo := ratesmocks.NewMockPriceReader(ctrl)

	// Freeze time for tests
	oldNow := utils.NowFunc
	utils.NowFunc = func() time.Time { return fixed }
	t.Cleanup(func() { utils.NowFunc = oldNow })

	svc := NewService(coinRepo, priceRepo, slog.Default())
	return ctx, ctrl, coinRepo, priceRepo, svc
}

// ---- Tests for GetAllRates ----

func TestGetAllRates_SuccessMixed(t *testing.T) {
	fixed := time.Date(2025, 8, 12, 15, 0, 0, 0, time.UTC)
	ctx, ctrl, coinRepo, priceRepo, svc := setupSvc(t, fixed)
	defer ctrl.Finish()

	coins := []domain.Coin{{Symbol: "BTC"}, {Symbol: "ETH"}}
	coinRepo.EXPECT().GetAllCoins(gomock.Any()).Return(coins, nil)
	priceRepo.EXPECT().GetLatestPrice(gomock.Any(), "BTC").Return(&domain.Price{CoinSymbol: "BTC", Value: 100.0, Timestamp: fixed}, nil)
	priceRepo.EXPECT().GetLatestPrice(gomock.Any(), "ETH").Return(nil, errors.New("db error"))

	got, err := svc.GetAllRates(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 price returned, got %d", len(got))
	}
	if got[0].Symbol != "BTC" || got[0].Price != 100.0 {
		t.Fatalf("unexpected price: %+v", got[0])
	}
}

func TestGetAllRates_CoinRepoError(t *testing.T) {
	fixed := time.Date(2025, 8, 12, 15, 0, 0, 0, time.UTC)
	ctx, ctrl, coinRepo, _, svc := setupSvc(t, fixed)
	defer ctrl.Finish()

	coinRepo.EXPECT().GetAllCoins(gomock.Any()).Return(nil, errors.New("boom"))

	_, err := svc.GetAllRates(ctx)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestGetAllRates_SkipNilLatestPrice(t *testing.T) {
	fixed := time.Date(2025, 8, 12, 15, 0, 0, 0, time.UTC)
	ctx, ctrl, coinRepo, priceRepo, svc := setupSvc(t, fixed)
	defer ctrl.Finish()

	coins := []domain.Coin{{Symbol: "BTC"}}
	coinRepo.EXPECT().GetAllCoins(gomock.Any()).Return(coins, nil)
	priceRepo.EXPECT().GetLatestPrice(gomock.Any(), "BTC").Return(nil, nil)

	_, err := svc.GetAllRates(ctx)
	if err == nil || !errors.Is(err, domain.ErrPriceNotFound) {
		t.Fatalf("expected domain.ErrPriceNotFound, got %v", err)
	}
}

// ---- Tests for GetRateBySymbol ----

func TestGetRateBySymbol_CoinNotFound(t *testing.T) {
	fixed := time.Date(2025, 8, 12, 15, 0, 0, 0, time.UTC)
	ctx, ctrl, coinRepo, _, svc := setupSvc(t, fixed)
	defer ctrl.Finish()

	coinRepo.EXPECT().GetCoinBySymbol(gomock.Any(), "BTC").Return(nil, nil)

	_, err := svc.GetRateBySymbol(ctx, "BTC")
	if err == nil || !errors.Is(err, domain.ErrCoinNotFound) {
		t.Fatalf("expected domain.ErrCoinNotFound, got: %v", err)
	}
}

func TestGetRateBySymbol_MinMaxError(t *testing.T) {
	fixed := time.Date(2025, 8, 12, 15, 0, 0, 0, time.UTC)
	ctx, ctrl, coinRepo, priceRepo, svc := setupSvc(t, fixed)
	defer ctrl.Finish()

	coinRepo.EXPECT().GetCoinBySymbol(gomock.Any(), "BTC").Return(&domain.Coin{Symbol: "BTC"}, nil)
	priceRepo.EXPECT().GetMinAndMaxPrices(gomock.Any(), "BTC", fixed.Add(-24*time.Hour)).Return(domain.Price{}, domain.Price{}, errors.New("minmax failed"))

	_, err := svc.GetRateBySymbol(ctx, "BTC")
	if err == nil || !errors.Is(err, domain.ErrPriceNotFound) {
		t.Fatalf("expected domain.ErrPriceNotFound, got: %v", err)
	}
}

func TestGetRateBySymbol_OldPriceMissing(t *testing.T) {
	fixed := time.Date(2025, 8, 12, 15, 0, 0, 0, time.UTC)
	ctx, ctrl, coinRepo, priceRepo, svc := setupSvc(t, fixed)
	defer ctrl.Finish()

	coinRepo.EXPECT().GetCoinBySymbol(gomock.Any(), "BTC").Return(&domain.Coin{Symbol: "BTC"}, nil)
	priceRepo.EXPECT().GetMinAndMaxPrices(gomock.Any(), "BTC", fixed.Add(-24*time.Hour)).Return(
		domain.Price{CoinSymbol: "BTC", Value: 90, Timestamp: fixed.Add(-23 * time.Hour)},
		domain.Price{CoinSymbol: "BTC", Value: 110, Timestamp: fixed.Add(-2 * time.Hour)},
		nil,
	)
	priceRepo.EXPECT().GetPriceBefore(gomock.Any(), "BTC", fixed.Add(-1*time.Hour)).Return(nil, nil)

	_, err := svc.GetRateBySymbol(ctx, "BTC")
	if err == nil || !errors.Is(err, domain.ErrPriceNotFound) {
		t.Fatalf("expected domain.ErrPriceNotFound, got: %v", err)
	}
}

func TestGetRateBySymbol_OldPriceZero(t *testing.T) {
	fixed := time.Date(2025, 8, 12, 15, 0, 0, 0, time.UTC)
	ctx, ctrl, coinRepo, priceRepo, svc := setupSvc(t, fixed)
	defer ctrl.Finish()

	coinRepo.EXPECT().GetCoinBySymbol(gomock.Any(), "BTC").Return(&domain.Coin{Symbol: "BTC"}, nil)
	priceRepo.EXPECT().GetMinAndMaxPrices(gomock.Any(), "BTC", fixed.Add(-24*time.Hour)).Return(
		domain.Price{CoinSymbol: "BTC", Value: 90, Timestamp: fixed.Add(-23 * time.Hour)},
		domain.Price{CoinSymbol: "BTC", Value: 110, Timestamp: fixed.Add(-2 * time.Hour)},
		nil,
	)
	priceRepo.EXPECT().GetPriceBefore(gomock.Any(), "BTC", fixed.Add(-1*time.Hour)).Return(&domain.Price{CoinSymbol: "BTC", Value: 0, Timestamp: fixed.Add(-1 * time.Hour)}, nil)

	_, err := svc.GetRateBySymbol(ctx, "BTC")
	if err == nil || !errors.Is(err, domain.ErrPriceNotFound) {
		t.Fatalf("expected domain.ErrPriceNotFound, got: %v", err)
	}
}

func TestGetRateBySymbol_CurrentMissing(t *testing.T) {
	fixed := time.Date(2025, 8, 12, 15, 0, 0, 0, time.UTC)
	ctx, ctrl, coinRepo, priceRepo, svc := setupSvc(t, fixed)
	defer ctrl.Finish()

	coinRepo.EXPECT().GetCoinBySymbol(gomock.Any(), "BTC").Return(&domain.Coin{Symbol: "BTC"}, nil)
	priceRepo.EXPECT().GetMinAndMaxPrices(gomock.Any(), "BTC", fixed.Add(-24*time.Hour)).Return(
		domain.Price{CoinSymbol: "BTC", Value: 90, Timestamp: fixed.Add(-23 * time.Hour)},
		domain.Price{CoinSymbol: "BTC", Value: 110, Timestamp: fixed.Add(-2 * time.Hour)},
		nil,
	)
	priceRepo.EXPECT().GetPriceBefore(gomock.Any(), "BTC", fixed.Add(-1*time.Hour)).Return(&domain.Price{CoinSymbol: "BTC", Value: 100, Timestamp: fixed.Add(-1 * time.Hour)}, nil)
	priceRepo.EXPECT().GetLatestPrice(gomock.Any(), "BTC").Return(nil, nil)

	_, err := svc.GetRateBySymbol(ctx, "BTC")
	if err == nil || !errors.Is(err, domain.ErrPriceNotFound) {
		t.Fatalf("expected domain.ErrPriceNotFound, got: %v", err)
	}
}

func TestGetRateBySymbol_Success(t *testing.T) {
	fixed := time.Date(2025, 8, 12, 15, 0, 0, 0, time.UTC)
	ctx, ctrl, coinRepo, priceRepo, svc := setupSvc(t, fixed)
	defer ctrl.Finish()

	coin := domain.Coin{Symbol: "BTC"}
	minP := domain.Price{CoinSymbol: "BTC", Value: 90, Timestamp: fixed.Add(-23 * time.Hour)}
	maxP := domain.Price{CoinSymbol: "BTC", Value: 110, Timestamp: fixed.Add(-2 * time.Hour)}
	old := &domain.Price{CoinSymbol: "BTC", Value: 100, Timestamp: fixed.Add(-1 * time.Hour)}
	cur := &domain.Price{CoinSymbol: "BTC", Value: 105, Timestamp: fixed}

	coinRepo.EXPECT().GetCoinBySymbol(gomock.Any(), "BTC").Return(&coin, nil)
	priceRepo.EXPECT().GetMinAndMaxPrices(gomock.Any(), "BTC", fixed.Add(-24*time.Hour)).Return(minP, maxP, nil)
	priceRepo.EXPECT().GetPriceBefore(gomock.Any(), "BTC", fixed.Add(-1*time.Hour)).Return(old, nil)
	priceRepo.EXPECT().GetLatestPrice(gomock.Any(), "BTC").Return(cur, nil)

	got, err := svc.GetRateBySymbol(ctx, "BTC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Min24h != minP.Value || got.Max24h != maxP.Value {
		t.Fatalf("unexpected min/max: got (%v, %v)", got.Min24h, got.Max24h)
	}
	if got.Price != cur.Value {
		t.Fatalf("unexpected price: got %v", got.Price)
	}
	if got.Symbol != "BTC" {
		t.Fatalf("unexpected symbol: got %v", got.Symbol)
	}
	expectedChange := 5.0 // (105-100)/100 * 100
	if diff := got.Change1hPct - expectedChange; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("unexpected changePct: got %v want %v", got.Change1hPct, expectedChange)
	}
	if !got.UpdatedAt.Equal(fixed) {
		t.Fatalf("unexpected UpdatedAt: got %v want %v", got.UpdatedAt, fixed)
	}
}
