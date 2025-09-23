package errors

import "errors"

var (
	ErrCoinNotFound  = errors.New("coin not found")
	ErrPriceNotFound = errors.New("price not found")
	ErrInternal      = errors.New("internal error")
)
