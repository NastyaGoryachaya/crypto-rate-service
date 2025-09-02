package bot

import (
	"fmt"
	"time"
)

// formatRateLine — короткая строка для рассылок (scheduler)
func formatRateLine(r RateDTO) string {
	// Если нет значений для расчета статистики, только текущее
	if r.Min24h == 0 && r.Max24h == 0 && r.Change1h == 0 {
		return fmt.Sprintf("%s | Текущая цена: %s",
			r.Symbol,
			humanPrice(r.Price),
		)
	}
	return fmt.Sprintf("%s | Текущая ценв: %s | Минимальная за 24ч: %s | Максимальная за 24ч: %s | Процент: %+.2f%%",
		r.Symbol,
		humanPrice(r.Price),
		humanPrice(r.Min24h),
		humanPrice(r.Max24h),
		r.Change1h,
	)
}

// formatRateDetails — подробное сообщение для команды /rates {symbol}
func formatRateDetails(r RateDTO) string {
	return fmt.Sprintf(
		"[%s]\nТекущая цена: %s\nМинимальная за 24ч: %s\nМаксимальная за 24ч: %s\nПроцент: %+.2f%%\nОбновлено: %s",
		r.Symbol,
		humanPrice(r.Price),
		humanPrice(r.Min24h),
		humanPrice(r.Max24h),
		r.Change1h,
		r.UpdatedAt.Format(time.RFC3339),
	)
}

// humanPrice — форматирование числа с двумя знаками после запятой.
func humanPrice(v float64) string {
	return fmt.Sprintf("%.2f", v)
}
