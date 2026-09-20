package domain

import "testing"

func TestCurrencyValid(t *testing.T) {
	cases := []struct {
		name string
		c    Currency
		want bool
	}{
		{"rub", CurrencyRUB, true},
		{"eur", CurrencyEUR, true},
		{"usd", CurrencyUSD, true},
		{"unknown code", Currency("GBP"), false},
		{"lowercase", Currency("rub"), false},
		{"empty", Currency(""), false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.c.Valid(); got != c.want {
				t.Fatalf("Valid(%q) = %v, want %v", c.c, got, c.want)
			}
		})
	}
}

// TestTierString: tier labels, including the sentinel and the fallback.
func TestTierString(t *testing.T) {
	cases := []struct {
		tier Tier
		want string
	}{
		{TierNone, "none"},
		{TierMinimal, "minimal"},
		{TierAcceptable, "acceptable"},
		{TierOptimal, "optimal"},
		{Tier(99), "unknown"},
	}
	for _, c := range cases {
		if got := c.tier.String(); got != c.want {
			t.Errorf("Tier(%d).String() = %q, want %q", c.tier, got, c.want)
		}
	}
}

// TestCurrencyExponent: two digits for every known currency and for an unknown one, so
// formatting never divides by the wrong power.
func TestCurrencyExponent(t *testing.T) {
	for _, c := range []Currency{CurrencyRUB, CurrencyEUR, CurrencyUSD, Currency("XXX")} {
		if got := c.Exponent(); got != 2 {
			t.Errorf("Exponent(%q) = %d, want 2", c, got)
		}
	}
}
