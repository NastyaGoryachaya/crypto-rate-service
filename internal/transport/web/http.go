package web

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"log/slog"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/consts"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/domain"
	errs "github.com/NastyaGoryachaya/crypto-rate-service/internal/errors"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/interfaces"
	"github.com/labstack/echo/v4"
)

type APIRate struct {
	Symbol      string    `json:"symbol"`
	Price       float64   `json:"price"`
	Min24h      *float64  `json:"min_24h,omitempty"`
	Max24h      *float64  `json:"max_24h,omitempty"`
	Change1hPct *float64  `json:"change_1h_pct,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToAPI — локальный конвертер транспорта
func ToAPI(item domain.Coin) APIRate {
	return APIRate{
		Symbol:    item.Symbol,
		Price:     item.Price,
		UpdatedAt: item.UpdatedAt,
	}
}

func ToAPIWithStats(latest domain.Coin, min, max, pct float64) APIRate {
	return APIRate{
		Symbol:      latest.Symbol,
		Price:       latest.Price,
		Min24h:      &min,
		Max24h:      &max,
		Change1hPct: &pct,
		UpdatedAt:   latest.UpdatedAt,
	}
}

// RatesHandler — HTTP‑handler для курсов.
type RatesHandler struct {
	logger  *slog.Logger
	svc     interfaces.Service
	timeout time.Duration
}

func NewRatesHandler(logger *slog.Logger, svc interfaces.Service, timeout time.Duration) *RatesHandler {
	if logger == nil {
		log.Fatal("nil logger")
	}
	if svc == nil {
		log.Fatal("nil service")
	}
	// Задаём таймаут по умолчанию, если он не задан
	if timeout <= 0 {
		timeout = time.Second * 3
	}
	return &RatesHandler{
		logger:  logger,
		svc:     svc,
		timeout: timeout,
	}
}

func (h *RatesHandler) RegisterRoutes(r interface {
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}) {
	// Регистрируем маршруты
	r.GET("/rates", h.GetRates)
	r.GET("/rates/:symbol", h.GetRateBySymbol)
}

func (h *RatesHandler) GetRates(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	items, err := h.svc.GetLatest(ctx)
	// Обрабатываем ошибки сервиса и возвращаем JSON
	if err == nil && len(items) == 0 {
		return c.JSON(http.StatusOK, []APIRate{})
	}
	if err != nil {
		if errors.Is(err, errs.ErrPriceNotFound) {
			return c.JSON(http.StatusNotFound, echo.Map{
				"error": "prices_not_found",
			})
		}
		h.logger.Error("GetAllRates failed",
			slog.String("op", "GetRates"),
			slog.String("error", err.Error()),
		)
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "internal_server_error",
		})
	}

	out := make([]APIRate, 0, len(items))
	for _, item := range items {
		out = append(out, ToAPI(item))
	}

	return c.JSON(http.StatusOK, out)
}

func (h *RatesHandler) GetRateBySymbol(c echo.Context) error {
	symbol := strings.ToUpper(strings.TrimSpace(c.Param("symbol")))
	if symbol == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "symbol_required",
		})
	}

	// Проверяем, поддерживается ли символ
	if !consts.IsTracked(symbol) {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":  "unsupported_symbol",
			"symbol": symbol,
		})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	now := time.Now().UTC()
	from := now.Add(-24 * time.Hour)

	latest, minV, maxV, pct, err := h.svc.GetLatestBySymbol(ctx, symbol, from, now)
	if err != nil {
		if errors.Is(err, errs.ErrCoinNotFound) {
			return c.JSON(http.StatusNotFound, echo.Map{
				"error":  "coin_not_found",
				"symbol": symbol,
			})
		} else if errors.Is(err, errs.ErrPriceNotFound) {
			return c.JSON(http.StatusNotFound, echo.Map{
				"error":  "prices_not_found",
				"symbol": symbol,
			})
		}
		h.logger.Error("GetRateBySymbol failed",
			slog.String("op", "GetRateBySymbol"),
			slog.String("symbol", symbol),
			slog.String("error", err.Error()),
		)
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "internal_server_error",
		})
	}

	out := ToAPIWithStats(latest, minV, maxV, pct)
	return c.JSON(http.StatusOK, out)
}
