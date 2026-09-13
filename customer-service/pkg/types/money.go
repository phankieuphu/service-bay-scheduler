package types

import (
	"strings"

	"github.com/shopspring/decimal"
)

func MoneyStringToDecimal(value string) decimal.Decimal {
	clean := strings.ReplaceAll(value, ",", "")

	d, err := decimal.NewFromString(clean)
	if err != nil {
		return decimal.Zero
	}

	return d
}

func DecimalToMoneyString(value decimal.Decimal) string {
	return value.String()
}
