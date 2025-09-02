package domain

import "time"

// Coin - представляет криптовалюту
type Coin struct {
	Symbol string `json:"symbol"` // BTC, ETH
}

// Price - хранит историческую цену монеты
type Price struct {
	CoinSymbol string    `json:"coin_symbol"` // Символ монеты (BTC, ETH)
	Value      float64   `json:"value"`       // Текущая цена
	Timestamp  time.Time `json:"timestamp"`   // Время получения курса (UTC)
}
