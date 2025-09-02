package rates

import "errors"

var (
	ErrCoinNotFound  = errors.New("coin not found")
	ErrPriceNotFound = errors.New("price not found")
	ErrMinMaxPrice   = errors.New("min/max price calculation error")
	ErrHourAgoPrice  = errors.New("hour-ago price unavailable")
)
