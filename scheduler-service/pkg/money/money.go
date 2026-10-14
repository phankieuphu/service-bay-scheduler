// Package money converts the catalog's decimal fees to the integer minor
// units billing-service and Stripe use.
package money

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// Stripe's zero- and three-decimal currencies; everything else has 2. Must
// match billing-service's domain.money.Money.
var (
	zeroDecimal  = map[string]bool{"BIF": true, "CLP": true, "DJF": true, "GNF": true, "JPY": true, "KMF": true, "KRW": true, "MGA": true, "PYG": true, "RWF": true, "UGX": true, "VND": true, "VUV": true, "XAF": true, "XOF": true, "XPF": true}
	threeDecimal = map[string]bool{"BHD": true, "JOD": true, "KWD": true, "OMR": true, "TND": true}
)

// Decimals is how many minor-unit decimals currency has, in Stripe's convention.
func Decimals(currency string) int32 {
	code := strings.ToUpper(currency)
	switch {
	case zeroDecimal[code]:
		return 0
	case threeDecimal[code]:
		return 3
	default:
		return 2
	}
}

// ToMinor converts amount to minor units of currency: 45.00 USD is 4500,
// 150000 VND is 150000. An amount with more precision than the currency has
// is an error rather than being rounded.
func ToMinor(amount decimal.Decimal, currency string) (int64, error) {
	shifted := amount.Shift(Decimals(currency))
	if !shifted.Equal(shifted.Truncate(0)) {
		return 0, fmt.Errorf("%s has more decimals than %s allows", amount.String(), strings.ToUpper(currency))
	}
	return shifted.IntPart(), nil
}
