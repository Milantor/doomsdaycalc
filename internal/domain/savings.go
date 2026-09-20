// Package domain: core business logic and types.
// No Telegram/SQL/network/etc: only functions and plain structs. Time is
// passed in, avoid using time.Now(). Money is int64 minor units.
// External dependencies are interfaces declared here.
// Dependencies point one way: bot -> service -> domain <- storage.
package domain

import (
	"context"
	"time"
)

// Currency: ISO-4217 code. Stored on the owning goal record.
type Currency string

const (
	CurrencyRUB Currency = "RUB"
	CurrencyEUR Currency = "EUR"
	CurrencyUSD Currency = "USD"
)

// Currencies: every currency the domain knows.
var Currencies = []Currency{CurrencyRUB, CurrencyEUR, CurrencyUSD}

// Exponent: number of decimal digits in one minor unit, e.g. 2 for RUB. Needed to
// format Money, since a raw integer has no decimal point.
func (c Currency) Exponent() int {
	switch c {
	case CurrencyRUB, CurrencyEUR, CurrencyUSD:
		return 2
	default:
		return 2
	}
}

// Valid: true when the code is in Currencies. Scans the set, so adding a currency
// is one constant plus one slice entry.
func (c Currency) Valid() bool {
	for _, known := range Currencies {
		if c == known {
			return true
		}
	}
	return false
}

// Money: amount in minor units (kopecks/cents).
// Owning record sets currency:
//   - Goal.Targets: Goal.Currency (what we save for, e.g. EUR);
//   - Deposit.Amount: Goal.SavingsCurrency (what we save in, e.g. RUB).
type Money int64

// Tier: minimal, acceptable or optimal saving target.
type Tier int

// TierNone: no tier reached yet. Sorts below TierMinimal.
const TierNone Tier = -1

const (
	TierMinimal Tier = iota
	TierAcceptable
	TierOptimal
)

// Tiers: every tier in ascending order.
var Tiers = []Tier{TierMinimal, TierAcceptable, TierOptimal}

// String: tier label.
func (t Tier) String() string {
	switch t {
	case TierNone:
		return "none"
	case TierMinimal:
		return "minimal"
	case TierAcceptable:
		return "acceptable"
	case TierOptimal:
		return "optimal"
	default:
		return "unknown"
	}
}

// Goal: a saving target with a deadline.
// Two currencies are separate because common case is "goal in EUR, save in RUB":
// Currency is what Targets are in, SavingsCurrency is what deposits are in.
// Calculator converts between them when they differ. Deadline is UTC; Timezone
// (IANA name, e.g. "Europe/Moscow") sets what "today" and "end of month" mean for
// a user.
type Goal struct {
	ID       int64
	UserID   int64
	Title    string
	Deadline time.Time
	Timezone string

	Currency        Currency // currency of Targets, e.g. EUR
	SavingsCurrency Currency // currency of deposits, e.g. RUB

	Targets   map[Tier]Money
	Active    bool
	CreatedAt time.Time
}

// Deposit: signed amount into (+) or out of (-) goals savings, in goals
// SavingsCurrency. Withdrawals are negative rows in same table.
type Deposit struct {
	ID         int64
	GoalID     int64
	Amount     Money
	HappenedAt time.Time
	Note       string
}

// GoalRepository: stores goals. Implemented in storage/postgres.
type GoalRepository interface {
	Create(ctx context.Context, g Goal) (Goal, error)
	Get(ctx context.Context, id int64) (Goal, error)
	ListByUser(ctx context.Context, userID int64) ([]Goal, error)
	// TODO: i need that? no caller yet, kept for goal editing.
	SetActive(ctx context.Context, id int64, active bool) error
}

// DepositRepository: stores deposits.
type DepositRepository interface {
	Add(ctx context.Context, d Deposit) (Deposit, error)
	ListByGoal(ctx context.Context, goalID int64) ([]Deposit, error)
}

// ComputeStatus, progress calculation, is in status.go with its tests.
