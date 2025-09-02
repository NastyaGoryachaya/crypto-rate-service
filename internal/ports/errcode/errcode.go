package errcode

type Code string

const (
	NotFoundCoins  Code = "NOT_FOUND_COINS"
	NotFoundPrices Code = "NOT_FOUND_PRICES"

	MinMaxPrice  Code = "MINMAX_PRICE"
	HourAgoPrice Code = "HOUR_AGO_PRICE"

	BadRequest Code = "BAD_REQUEST"
	Internal   Code = "INTERNAL_ERROR"
)
