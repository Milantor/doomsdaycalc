package bot

import (
	"testing"
	"time"

	"lab042.ru/doomsdaycalc/internal/domain"
	"lab042.ru/doomsdaycalc/internal/i18n"
)

// seededStatusGoal: a goal with a controlled window and targets, so status numbers are
// predictable in tests without reading the clock.
func seededStatusGoal(id, userID int64) domain.Goal {
	return domain.Goal{
		ID:              id,
		UserID:          userID,
		Title:           "Trip",
		CreatedAt:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Deadline:        time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC),
		Timezone:        "UTC",
		Currency:        domain.CurrencyRUB,
		SavingsCurrency: domain.CurrencyRUB,
		Targets: map[domain.Tier]domain.Money{
			domain.TierMinimal:    100000,
			domain.TierAcceptable: 200000,
			domain.TierOptimal:    300000,
		},
		Active: true,
	}
}

// TestFormatStatus: one deposit, five days left, six days of a ten day window counted.
// Every tier is behind by its arrears, so the pace line carries the currency.
func TestFormatStatus(t *testing.T) {
	m := i18n.Get(domain.LangEN)
	g := seededStatusGoal(7, 100)
	now := time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC)
	deposits := []domain.Deposit{{GoalID: 7, Amount: 50000, HappenedAt: now}}
	st := domain.ComputeStatus(g, deposits, domain.IdentityRate(), now)

	want := `🎯 Trip
Saved: 500 of 1000/2000/3000 RUB (minimal/acceptable/optimal)
Left: 500/1500/2500 RUB
Per day: 100/300/500 RUB
Per month: 500/1500/2500 RUB
Deadline: 2024-01-11 · 5 days left
Pace: 100/700/1300 RUB`

	if got := formatStatus(g, st, m); got != want {
		t.Fatalf("formatStatus mismatch.\n got: %q\nwant: %q", got, want)
	}
}

// TestFormatStatusAllOnPace: every tier on pace, so the pace line holds checks only and
// drops the currency.
func TestFormatStatusAllOnPace(t *testing.T) {
	m := i18n.Get(domain.LangEN)
	g := seededStatusGoal(7, 100)
	now := time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC)
	deposits := []domain.Deposit{{GoalID: 7, Amount: 250000, HappenedAt: now}}
	st := domain.ComputeStatus(g, deposits, domain.IdentityRate(), now)

	want := `🎯 Trip
Saved: 2500 of ✅/✅/3000 RUB (minimal/acceptable/optimal)
Left: 0/0/500 RUB
Per day: 0/0/100 RUB
Per month: 0/0/500 RUB
Deadline: 2024-01-11 · 5 days left
Pace: ✅/✅/✅`

	if got := formatStatus(g, st, m); got != want {
		t.Fatalf("formatStatus mismatch.\n got: %q\nwant: %q", got, want)
	}
}

// TestFormatStatusPartialChecks: the minimal tier reached turns into a check in the saved
// line, the open tiers keep their numbers.
func TestFormatStatusPartialChecks(t *testing.T) {
	m := i18n.Get(domain.LangEN)
	g := seededStatusGoal(7, 100)
	now := time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC)
	deposits := []domain.Deposit{{GoalID: 7, Amount: 150000, HappenedAt: now}}
	st := domain.ComputeStatus(g, deposits, domain.IdentityRate(), now)

	want := `🎯 Trip
Saved: 1500 of ✅/2000/3000 RUB (minimal/acceptable/optimal)
Left: 0/500/1500 RUB
Per day: 0/100/300 RUB
Per month: 0/500/1500 RUB
Deadline: 2024-01-11 · 5 days left
Pace: ✅/✅/300 RUB`

	if got := formatStatus(g, st, m); got != want {
		t.Fatalf("formatStatus mismatch.\n got: %q\nwant: %q", got, want)
	}
}

// TestFormatStatusGoalDone: every tier reached switches to the short done view.
func TestFormatStatusGoalDone(t *testing.T) {
	m := i18n.Get(domain.LangEN)
	g := seededStatusGoal(7, 100)
	now := time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC)
	deposits := []domain.Deposit{{GoalID: 7, Amount: 300000, HappenedAt: now}}
	st := domain.ComputeStatus(g, deposits, domain.IdentityRate(), now)

	want := `🎯 Trip
3000 RUB
Goal reached!
Deadline: 2024-01-11`

	if got := formatStatus(g, st, m); got != want {
		t.Fatalf("formatStatus mismatch.\n got: %q\nwant: %q", got, want)
	}
}
