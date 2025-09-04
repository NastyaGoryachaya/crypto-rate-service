package domain

import "time"

// Coin - представляет криптовалюту
type Coin struct {
	Symbol string // BTC, ETH
}

// Price - хранит историческую цену монеты
type Price struct {
	CoinSymbol string    // Символ монеты (BTC, ETH)
	Value      float64   // Текущая цена
	Timestamp  time.Time // Время получения курса (UTC)
}
