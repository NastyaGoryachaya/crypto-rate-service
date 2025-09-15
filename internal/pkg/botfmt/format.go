package botfmt

import (
	"fmt"
	"math"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/domain"
)

// FormatRateLine — короткая строка для рассылок /rates
func FormatRateLine(r domain.Coin) string {
	return fmt.Sprintf("%s | Текущая цена: %s | Обновлено: %s",
		r.Symbol,
		humanPrice(r.Price),
		r.UpdatedAt.Format("15:04:05"),
	)
}

// FormatRateDetails — подробное сообщение для команды /rates {symbol}
func FormatRateDetails(latest domain.Coin, min, max, pct float64) string {
	msg := fmt.Sprintf("Изменение за 1ч: %+.2f%%", pct)
	// Добавляем пометку, если по отображению это 0.00%
	if math.Abs(pct) < 0.005 {
		msg += " (набираем данные для расчёта)"
	}

	return fmt.Sprintf(
		"[%s]\nТекущая цена: %s\nМинимальная за 24ч: %s\nМаксимальная за 24ч: %s\n%s\nОбновлено: %s",
		latest.Symbol,
		humanPrice(latest.Price),
		humanPrice(min),
		humanPrice(max),
		msg,
		latest.UpdatedAt.Format("15:04:05"),
	)
}

// humanPrice — форматирование числа с двумя знаками после запятой.
func humanPrice(v float64) string {
	return fmt.Sprintf("%.2f", v)
}
