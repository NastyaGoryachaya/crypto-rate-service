package adapter

import (
	"context"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/bot"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/service/rates"
)

// serviceRatesReader — адаптер, который превращает сервис курсов в интерфейс бота RatesReader.

type serviceRatesReader struct{ svc rates.Service }

// NewRatesReader — конструктор адаптера над сервисом курсов.
func NewRatesReader(svc rates.Service) bot.RatesReader {
	return serviceRatesReader{svc: svc}
}

// GetCurrencyRates — возвращает список курсов для всех монет, преобразуя их в DTO бота.
func (a serviceRatesReader) GetCurrencyRates(ctx context.Context) ([]bot.RateDTO, error) {
	items, err := a.svc.GetAllRates(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]bot.RateDTO, 0, len(items))
	for _, it := range items {
		out = append(out, bot.RateDTO{
			Symbol:    it.Symbol,
			Price:     it.Price,
			Min24h:    it.Min24h,
			Max24h:    it.Max24h,
			Change1h:  it.Change1hPct,
			UpdatedAt: it.UpdatedAt,
		})
	}
	return out, nil
}

// GetCurrencyRateBySymbol — возвращает курс по конкретному символу в формате DTO бота.
func (a serviceRatesReader) GetCurrencyRateBySymbol(ctx context.Context, symbol string) (bot.RateDTO, error) {
	it, err := a.svc.GetRateBySymbol(ctx, symbol)
	if err != nil {
		return bot.RateDTO{}, err
	}
	return bot.RateDTO{
		Symbol:    it.Symbol,
		Price:     it.Price,
		Min24h:    it.Min24h,
		Max24h:    it.Max24h,
		Change1h:  it.Change1hPct,
		UpdatedAt: it.UpdatedAt,
	}, nil
}
