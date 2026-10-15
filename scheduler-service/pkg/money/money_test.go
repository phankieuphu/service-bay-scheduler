package money

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestToMinor(t *testing.T) {
	cases := []struct {
		amount, currency string
		want             int64
		wantErr          bool
	}{
		{"45.00", "USD", 4500, false},
		{"60.5", "usd", 6050, false},
		{"150000", "VND", 150000, false},
		{"1.5", "VND", 0, true},
		{"1.005", "USD", 0, true},
		{"1.234", "KWD", 1234, false},
	}
	for _, c := range cases {
		got, err := ToMinor(decimal.RequireFromString(c.amount), c.currency)
		if (err != nil) != c.wantErr || got != c.want {
			t.Errorf("ToMinor(%s, %s) = %d, %v; want %d, err=%v", c.amount, c.currency, got, err, c.want, c.wantErr)
		}
	}
}
