package httptransport

import (
	"errors"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/ports/errcode"
	errors2 "github.com/NastyaGoryachaya/crypto-rate-service/internal/service/rates"
)

func FromServiceError(err error) errcode.Code {
	switch {
	case errors.Is(err, errors2.ErrCoinNotFound):
		return errcode.NotFoundCoins
	case errors.Is(err, errors2.ErrPriceNotFound),
		errors.Is(err, errors2.ErrMinMaxPrice),
		errors.Is(err, errors2.ErrHourAgoPrice):
		return errcode.NotFoundPrices
	default:
		return errcode.Internal
	}
}
