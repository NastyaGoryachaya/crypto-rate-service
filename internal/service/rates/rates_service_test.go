package rates

import (
	"context"
	"errors"
	"testing"
	"time"

	"log/slog"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/domain"
	derrors "github.com/NastyaGoryachaya/crypto-rate-service/internal/errors"
	ratesmocks "github.com/NastyaGoryachaya/crypto-rate-service/internal/service/rates/mocks"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5"
)

// helper to build service with mocks
func setupSvc(t *testing.T) (context.Context, *gomock.Controller, *ratesmocks.MockStorage, *ratesmocks.MockCryptoProvider, *Service) {
	t.Helper()
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	storage := ratesmocks.NewMockStorage(ctrl)
	provider := ratesmocks.NewMockCryptoProvider(ctrl)
	svc := NewService(storage, provider, slog.Default())
	return ctx, ctrl, storage, provider, svc
}

// -------------------------
// GetLatest
// -------------------------

func TestGetLatest_Success(t *testing.T) {
	ctx, ctrl, storage, _, svc := setupSvc(t)
	defer ctrl.Finish()

	now := time.Date(2025, 9, 1, 12, 0, 0, 0, time.UTC)
	in := []domain.Coin{
		{Symbol: "BTC", Price: 100.0, UpdatedAt: now},
		{Symbol: "ETH", Price: 200.0, UpdatedAt: now},
	}
	storage.EXPECT().GetAllCoins(gomock.Any()).Return(in, nil)

	got, err := svc.GetLatest(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 coins, got %d", len(got))
	}
	if got[0].Symbol != "BTC" || got[1].Symbol != "ETH" {
		t.Fatalf("unexpected symbols: %+v", got)
	}
}

func TestGetLatest_RepoError(t *testing.T) {
	ctx, ctrl, storage, _, svc := setupSvc(t)
	defer ctrl.Finish()

	storage.EXPECT().GetAllCoins(gomock.Any()).Return(nil, errors.New("db down"))

	_, err := svc.GetLatest(ctx)
	if err == nil || !errors.Is(err, derrors.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", err)
	}
}

func TestGetLatest_Empty(t *testing.T) {
	ctx, ctrl, storage, _, svc := setupSvc(t)
	defer ctrl.Finish()

	storage.EXPECT().GetAllCoins(gomock.Any()).Return([]domain.Coin{}, nil)

	_, err := svc.GetLatest(ctx)
	if err == nil || !errors.Is(err, derrors.ErrPriceNotFound) {
		t.Fatalf("expected ErrPriceNotFound, got %v", err)
	}
}

// -------------------------
// GetLatestBySymbol
// -------------------------

func TestGetLatestBySymbol_CoinNotFound(t *testing.T) {
	ctx, ctrl, storage, _, svc := setupSvc(t)
	defer ctrl.Finish()

	from := time.Date(2025, 9, 1, 10, 0, 0, 0, time.UTC)
	to := time.Date(2025, 9, 1, 12, 0, 0, 0, time.UTC)

	storage.EXPECT().GetCoinBySymbol(gomock.Any(), "BTC").Return(domain.Coin{}, pgx.ErrNoRows)

	_, _, _, _, err := svc.GetLatestBySymbol(ctx, "BTC", from, to)
	if err == nil || !errors.Is(err, derrors.ErrCoinNotFound) {
		t.Fatalf("expected ErrCoinNotFound, got %v", err)
	}
}

func TestGetLatestBySymbol_NoHistory(t *testing.T) {
	ctx, ctrl, storage, _, svc := setupSvc(t)
	defer ctrl.Finish()

	from := time.Date(2025, 9, 1, 10, 0, 0, 0, time.UTC)
	to := time.Date(2025, 9, 1, 12, 0, 0, 0, time.UTC)

	latest := domain.Coin{Symbol: "BTC", Price: 105, UpdatedAt: to}
	storage.EXPECT().GetCoinBySymbol(gomock.Any(), "BTC").Return(latest, nil)
	storage.EXPECT().History(gomock.Any(), "BTC", from, to).Return([]domain.Coin{}, nil)

	_, _, _, _, err := svc.GetLatestBySymbol(ctx, "BTC", from, to)
	if err == nil || !errors.Is(err, derrors.ErrPriceNotFound) {
		t.Fatalf("expected ErrPriceNotFound, got %v", err)
	}
}

func TestGetLatestBySymbol_InternalHistoryError(t *testing.T) {
	ctx, ctrl, storage, _, svc := setupSvc(t)
	defer ctrl.Finish()

	from := time.Date(2025, 9, 1, 10, 0, 0, 0, time.UTC)
	to := time.Date(2025, 9, 1, 12, 0, 0, 0, time.UTC)

	latest := domain.Coin{Symbol: "BTC", Price: 105, UpdatedAt: to}
	storage.EXPECT().GetCoinBySymbol(gomock.Any(), "BTC").Return(latest, nil)
	storage.EXPECT().History(gomock.Any(), "BTC", from, to).Return(nil, errors.New("db error"))

	_, _, _, _, err := svc.GetLatestBySymbol(ctx, "BTC", from, to)
	if err == nil || !errors.Is(err, derrors.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", err)
	}
}

func TestGetLatestBySymbol_NoPrevForPct(t *testing.T) {
	ctx, ctrl, storage, _, svc := setupSvc(t)
	defer ctrl.Finish()

	from := time.Date(2025, 9, 1, 10, 0, 0, 0, time.UTC)
	to := time.Date(2025, 9, 1, 12, 0, 0, 0, time.UTC)

	latest := domain.Coin{Symbol: "BTC", Price: 105, UpdatedAt: to}
	history := []domain.Coin{
		{Symbol: "BTC", Price: 90, UpdatedAt: to.Add(-45 * time.Minute)},
		{Symbol: "BTC", Price: 110, UpdatedAt: to.Add(-30 * time.Minute)},
	}

	storage.EXPECT().GetCoinBySymbol(gomock.Any(), "BTC").Return(latest, nil)
	storage.EXPECT().History(gomock.Any(), "BTC", from, to).Return(history, nil)

	gotLatest, minPrice, maxPrice, pct, err := svc.GetLatestBySymbol(ctx, "BTC", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotLatest.Symbol != "BTC" || gotLatest.Price != 105 || !gotLatest.UpdatedAt.Equal(to) {
		t.Fatalf("unexpected latest: %+v", gotLatest)
	}
	if minPrice != 90 || maxPrice != 110 {
		t.Fatalf("unexpected min/max: (%v, %v)", minPrice, maxPrice)
	}
	// expected pct == 0 when there is no prev <= threshold
	if pct != 0 {
		t.Fatalf("expected pct == 0 when no prev <= threshold, got %v", pct)
	}
}

func TestGetLatestBySymbol_Success(t *testing.T) {
	ctx, ctrl, storage, _, svc := setupSvc(t)
	defer ctrl.Finish()

	from := time.Date(2025, 9, 1, 10, 0, 0, 0, time.UTC)
	to := time.Date(2025, 9, 1, 12, 0, 0, 0, time.UTC)

	latest := domain.Coin{Symbol: "BTC", Price: 105, UpdatedAt: to}
	// history covers min=90, max=110, and prev (<= to-1h) = 100
	history := []domain.Coin{
		{Symbol: "BTC", Price: 90, UpdatedAt: from.Add(1 * time.Hour)},    // min
		{Symbol: "BTC", Price: 100, UpdatedAt: to.Add(-1 * time.Hour)},    // prev (threshold)
		{Symbol: "BTC", Price: 110, UpdatedAt: to.Add(-30 * time.Minute)}, // max
	}

	storage.EXPECT().GetCoinBySymbol(gomock.Any(), "BTC").Return(latest, nil)
	storage.EXPECT().History(gomock.Any(), "BTC", from, to).Return(history, nil)

	gotLatest, minPrice, maxPrice, pct, err := svc.GetLatestBySymbol(ctx, "BTC", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotLatest.Symbol != "BTC" || gotLatest.Price != 105 || !gotLatest.UpdatedAt.Equal(to) {
		t.Fatalf("unexpected latest: %+v", gotLatest)
	}
	if minPrice != 90 || maxPrice != 110 {
		t.Fatalf("unexpected min/max: (%v, %v)", minPrice, maxPrice)
	}
	// expected pct = (105-100)/100*100 = 5
	if diff := pct - 5.0; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("unexpected pct: got %v want %v", pct, 5.0)
	}
}

// -------------------------
// FetchAndSaveCurrency
// -------------------------

func TestFetchAndSaveCurrency_ProviderError(t *testing.T) {
	ctx, ctrl, _, provider, svc := setupSvc(t)
	defer ctrl.Finish()

	provider.EXPECT().FetchRates(gomock.Any()).Return(nil, errors.New("provider down"))

	err := svc.FetchAndSaveCurrency(ctx)
	if err == nil || !errors.Is(err, derrors.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", err)
	}
}

func TestFetchAndSaveCurrency_SaveError(t *testing.T) {
	ctx, ctrl, storage, provider, svc := setupSvc(t)
	defer ctrl.Finish()

	now := time.Date(2025, 9, 1, 12, 0, 0, 0, time.UTC)
	items := []domain.Coin{
		{Symbol: "BTC", Price: 100, UpdatedAt: now},
	}
	provider.EXPECT().FetchRates(gomock.Any()).Return(items, nil)
	storage.EXPECT().SaveCoins(gomock.Any(), items).Return(errors.New("db write failed"))

	err := svc.FetchAndSaveCurrency(ctx)
	if err == nil || !errors.Is(err, derrors.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", err)
	}
}

func TestFetchAndSaveCurrency_Success(t *testing.T) {
	ctx, ctrl, storage, provider, svc := setupSvc(t)
	defer ctrl.Finish()

	now := time.Date(2025, 9, 1, 12, 0, 0, 0, time.UTC)
	items := []domain.Coin{
		{Symbol: "BTC", Price: 100, UpdatedAt: now},
		{Symbol: "ETH", Price: 200, UpdatedAt: now},
	}
	provider.EXPECT().FetchRates(gomock.Any()).Return(items, nil)
	storage.EXPECT().SaveCoins(gomock.Any(), items).Return(nil)

	if err := svc.FetchAndSaveCurrency(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
