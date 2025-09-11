package consts

import "strings"

var TrackedCoins = []string{"BTC", "ETH"}

func IsTracked(sym string) bool {
	s := strings.ToUpper(sym)
	for _, t := range TrackedCoins {
		if s == t {
			return true
		}
	}
	return false
}
