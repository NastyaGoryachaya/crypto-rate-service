package domain

import "time"

// Coin - представляет криптовалюту
type Coin struct {
	Symbol    string // BTC, ETH
	Price     float64
	UpdatedAt time.Time
}
