package domain

import (
	"errors"
	"testing"
)

func TestValidateGoal(t *testing.T) {
	now := utc(2024, 1, 1, 0)

	// valid: a goal every check passes. Tests copy it and break one thing.
	valid := func() Goal {
		g := mkGoal(CurrencyEUR, CurrencyRUB, now, utc(2024, 2, 1, 0), 10000, 20000, 30000)
		g.UserID = 1
		g.Title = "New laptop"
		return g
	}

	cases := []struct {
		name      string
		mutate    func(*Goal)
		wantValid bool
	}{
		{"valid", func(*Goal) {}, true},
		{"all three targets equal", func(g *Goal) {
			g.Targets[TierMinimal] = 5000
			g.Targets[TierAcceptable] = 5000
			g.Targets[TierOptimal] = 5000
		}, true},
		{"minimal equals acceptable", func(g *Goal) { g.Targets[TierMinimal] = 20000 }, true},
		{"acceptable equals optimal", func(g *Goal) { g.Targets[TierOptimal] = 20000 }, true},
		{"empty title", func(g *Goal) { g.Title = "" }, false},
		{"blank title", func(g *Goal) { g.Title = "   " }, false},
		{"deadline in past", func(g *Goal) { g.Deadline = utc(2023, 12, 31, 0) }, false},
		{"deadline equal now", func(g *Goal) { g.Deadline = now }, false},
		{"zero minimal target", func(g *Goal) { g.Targets[TierMinimal] = 0 }, false},
		{"negative target", func(g *Goal) { g.Targets[TierAcceptable] = -1 }, false},
		{"minimal above acceptable", func(g *Goal) { g.Targets[TierMinimal] = 25000 }, false},
		{"optimal below acceptable", func(g *Goal) { g.Targets[TierOptimal] = 10000 }, false},
		{"unknown currency", func(g *Goal) { g.Currency = Currency("GBP") }, false},
		{"lowercase currency", func(g *Goal) { g.Currency = Currency("rub") }, false},
		{"empty currency", func(g *Goal) { g.Currency = "" }, false},
		{"unknown savings currency", func(g *Goal) { g.SavingsCurrency = Currency("XX") }, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := valid()
			c.mutate(&g)

			err := ValidateGoal(g, now)
			if c.wantValid {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, ErrInvalidGoal) {
				t.Fatalf("err = %v, want ErrInvalidGoal", err)
			}
		})
	}
}
