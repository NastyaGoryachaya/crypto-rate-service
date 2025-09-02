package httptransport

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"log/slog"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/ports/errcode"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/service/rates"
	"github.com/labstack/echo/v4"
)

// Percent — число с плавающей точкой, представляющее процентное изменение (например, 0.123 = 12.3%).
// Кастомный JSON-маршалер выводит число с 3 знаками после запятой.
type Percent float64

func (p Percent) MarshalJSON() ([]byte, error) {
	v := float64(p)
	return []byte(strconv.FormatFloat(v, 'f', 3, 64)), nil
}

// RatesService — абстракция для работы с курсами.
type RatesService interface {
	GetAllRates(ctx context.Context) ([]rates.RateStats, error)
	GetRateBySymbol(ctx context.Context, symbol string) (rates.RateStats, error)
}

// Rate — DTO для ответа API с информацией о курсе.
type Rate struct {
	Symbol      string    `json:"symbol"`
	Price       float64   `json:"price"`
	Min24h      *float64  `json:"min_24h,omitempty"`
	Max24h      *float64  `json:"max_24h,omitempty"`
	Change1hPct *Percent  `json:"change_1h_pct,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func makeRate(item rates.RateStats) Rate {
	r := Rate{
		Symbol:    item.Symbol,
		Price:     item.Price,
		UpdatedAt: item.UpdatedAt,
	}
	// Добавляем min за 24 часа, если он есть
	if item.Min24h != 0 {
		v := item.Min24h
		r.Min24h = &v
	}
	// Добавляем max за 24 часа, если он есть
	if item.Max24h != 0 {
		v := item.Max24h
		r.Max24h = &v
	}
	// Добавляем изменение за 1 час в процентах, если оно есть
	if item.Change1hPct != 0 {
		p := Percent(item.Change1hPct)
		r.Change1hPct = &p
	}
	return r
}

// RatesHandler — HTTP‑handler для курсов.
type RatesHandler struct {
	logger  *slog.Logger
	svc     RatesService
	timeout time.Duration
}

func NewRatesHandler(logger *slog.Logger, svc RatesService, timeout time.Duration) *RatesHandler {
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

	items, err := h.svc.GetAllRates(ctx)
	// Обрабатываем ошибки сервиса и возвращаем JSON
	if err == nil && len(items) == 0 {
		return c.JSON(http.StatusOK, []Rate{})
	}
	if err != nil {
		code := FromServiceError(err)
		switch code {
		case errcode.NotFoundPrices:
			// Нет актуальных цен — 404
			return c.JSON(http.StatusNotFound, echo.Map{
				"error": "prices_not_found",
			})
		case errcode.BadRequest:
			return c.JSON(http.StatusBadRequest, echo.Map{
				"error": "bad_request",
			})
		default:
			h.logger.Error("GetAllRates failed",
				slog.String("op", "GetRates"),
				slog.String("error", err.Error()),
			)
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"error": "internal_server_error",
			})
		}
	}

	out := make([]Rate, 0, len(items))
	for _, item := range items {
		out = append(out, makeRate(item))
	}

	return c.JSON(http.StatusOK, out)
}

func (h *RatesHandler) GetRateBySymbol(c echo.Context) error {
	// Проверяем символ, обрабатываем ошибки и возвращаем JSON
	symbol := strings.ToUpper(strings.TrimSpace(c.Param("symbol")))
	if symbol == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "symbol_required",
		})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	item, err := h.svc.GetRateBySymbol(ctx, symbol)
	if err != nil {
		code := FromServiceError(err)
		switch code {
		case errcode.NotFoundCoins:
			return c.JSON(http.StatusNotFound, echo.Map{
				"error":  "coin_not_found",
				"symbol": symbol,
			})
		case errcode.NotFoundPrices:
			return c.JSON(http.StatusNotFound, echo.Map{
				"error":  "prices_not_found",
				"symbol": symbol,
			})
		case errcode.MinMaxPrice:
			return c.JSON(http.StatusUnprocessableEntity, echo.Map{
				"error":  "minmax_unavailable",
				"symbol": symbol,
			})
		case errcode.HourAgoPrice:
			return c.JSON(http.StatusUnprocessableEntity, echo.Map{
				"error":  "not_enough_data_for_change_1h",
				"symbol": symbol,
			})
		case errcode.BadRequest:
			return c.JSON(http.StatusBadRequest, echo.Map{
				"error": "bad_request",
			})
		default:
			h.logger.Error("GetRateBySymbol failed",
				slog.String("op", "GetRateBySymbol"),
				slog.String("symbol", symbol),
				slog.String("error", err.Error()),
			)
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"error": "internal_server_error",
			})
		}
	}

	out := makeRate(item)
	return c.JSON(http.StatusOK, out)
}
