package domain

import (
	"testing"
	"time"
)

func utc(y int, mo time.Month, d, h int) time.Time {
	return time.Date(y, mo, d, h, 0, 0, 0, time.UTC)
}

func mkGoal(currency, savings Currency, created, deadline time.Time, min, ok, max Money) Goal {
	return Goal{
		Currency:        currency,
		SavingsCurrency: savings,
		CreatedAt:       created,
		Deadline:        deadline,
		Timezone:        "UTC",
		Targets: map[Tier]Money{
			TierMinimal:    min,
			TierAcceptable: ok,
			TierOptimal:    max,
		},
	}
}

func TestExchangeRateConvert(t *testing.T) {
	cases := []struct {
		name string
		m    Money
		rate ExchangeRate
		want Money
	}{
		{"zero value is identity", 12345, ExchangeRate{}, 12345},
		{"explicit identity", 12345, IdentityRate(), 12345},
		{"halve", 1000, ExchangeRate{Num: 1, Den: 2}, 500},
		{"double", 1000, ExchangeRate{Num: 2, Den: 1}, 2000},
		{"round half up", 1, ExchangeRate{Num: 2, Den: 3}, 1}, // 0.667 -> 1
		{"round down", 1, ExchangeRate{Num: 1, Den: 3}, 0},    // 0.333 -> 0
		{"one hundredth", 50000, ExchangeRate{Num: 1, Den: 100}, 500},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.rate.Convert(c.m); got != c.want {
				t.Fatalf("Convert(%d) = %d, want %d", c.m, got, c.want)
			}
		})
	}
}

func TestDaysUntil(t *testing.T) {
	cases := []struct {
		name     string
		now      time.Time
		deadline time.Time
		want     int
	}{
		{"ten days", utc(2024, 1, 1, 0), utc(2024, 1, 11, 0), 10},
		{"partial day counts", utc(2024, 1, 1, 0), utc(2024, 1, 1, 5), 1},
		{"ten days and an hour rounds up", utc(2024, 1, 1, 0), utc(2024, 1, 11, 1), 11},
		{"past", utc(2024, 1, 1, 0), utc(2023, 12, 31, 0), 0},
		{"same instant", utc(2024, 1, 1, 0), utc(2024, 1, 1, 0), 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := daysUntil(c.now, c.deadline); got != c.want {
				t.Fatalf("daysUntil = %d, want %d", got, c.want)
			}
		})
	}
}

func TestMonthsUntil(t *testing.T) {
	cases := []struct {
		name     string
		now      time.Time
		deadline time.Time
		want     int
	}{
		{"same month is one", utc(2024, 1, 1, 0), utc(2024, 1, 11, 0), 1},
		{"two months", utc(2024, 1, 1, 0), utc(2024, 3, 1, 0), 2},
		{"whole month", utc(2024, 1, 15, 0), utc(2024, 2, 15, 0), 1},
		{"partial month is one", utc(2024, 1, 20, 0), utc(2024, 2, 15, 0), 1},
		{"three months", utc(2024, 1, 1, 0), utc(2024, 4, 1, 0), 3},
		{"past is zero", utc(2024, 1, 1, 0), utc(2023, 1, 1, 0), 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := monthsUntil(c.now, c.deadline, time.UTC); got != c.want {
				t.Fatalf("monthsUntil = %d, want %d", got, c.want)
			}
		})
	}
}

func TestComputeStatus(t *testing.T) {
	cases := []struct {
		name       string
		goal       Goal
		deposits   []Deposit
		rate       ExchangeRate
		now        time.Time
		saved      Money
		savedToday Money
		reached    Tier
		daysLeft   int
		monthsLeft int
		minRemain  Money
		minPerDay  Money
		minReached bool
		minOnTrack bool
		minBehind  Money
	}{
		{
			name:       "fresh goal, no deposits",
			goal:       mkGoal(CurrencyRUB, CurrencyRUB, utc(2024, 1, 1, 0), utc(2024, 1, 11, 0), 10000, 20000, 30000),
			now:        utc(2024, 1, 1, 0),
			saved:      0,
			savedToday: 0,
			reached:    TierNone,
			daysLeft:   10,
			monthsLeft: 1,
			minRemain:  10000,
			minPerDay:  1000,
			minReached: false,
			minOnTrack: true, // nothing expected yet
			minBehind:  0,
		},
		{
			name:       "acceptable reached",
			goal:       mkGoal(CurrencyRUB, CurrencyRUB, utc(2024, 1, 1, 0), utc(2024, 1, 11, 0), 10000, 20000, 30000),
			deposits:   []Deposit{{Amount: 25000, HappenedAt: utc(2024, 1, 1, 0)}},
			now:        utc(2024, 1, 1, 0),
			saved:      25000,
			savedToday: 25000,
			reached:    TierAcceptable,
			daysLeft:   10,
			monthsLeft: 1,
			minRemain:  0,
			minPerDay:  0,
			minReached: true,
			minOnTrack: true,
			minBehind:  0,
		},
		{
			name:       "deadline passed",
			goal:       mkGoal(CurrencyRUB, CurrencyRUB, utc(2024, 1, 1, 0), utc(2024, 1, 11, 0), 10000, 20000, 30000),
			now:        utc(2024, 2, 1, 0),
			saved:      0,
			savedToday: 0,
			reached:    TierNone,
			daysLeft:   0,
			monthsLeft: 0,
			minRemain:  10000,
			minPerDay:  10000, // everything is due today
			minReached: false,
			minOnTrack: false,
			minBehind:  10000, // whole target is owed
		},
		{
			name:       "goal in EUR, savings in RUB",
			goal:       mkGoal(CurrencyEUR, CurrencyRUB, utc(2024, 1, 1, 0), utc(2024, 1, 11, 0), 1000, 2000, 3000),
			deposits:   []Deposit{{Amount: 50000, HappenedAt: utc(2024, 1, 1, 0)}}, // 50000 RUB kopecks
			rate:       ExchangeRate{Num: 1, Den: 100},                             // 1 RUB kopeck = 0.01 EUR cent
			now:        utc(2024, 1, 1, 0),
			saved:      500,
			savedToday: 500,
			reached:    TierNone,
			daysLeft:   10,
			monthsLeft: 1,
			minRemain:  500,
			minPerDay:  50,
			minReached: false,
			minOnTrack: true,
			minBehind:  0,
		},
		{
			name:       "per-day rounds up",
			goal:       mkGoal(CurrencyRUB, CurrencyRUB, utc(2024, 1, 1, 0), utc(2024, 1, 4, 0), 10000, 20000, 30000),
			now:        utc(2024, 1, 1, 0),
			saved:      0,
			savedToday: 0,
			reached:    TierNone,
			daysLeft:   3,
			monthsLeft: 1,
			minRemain:  10000,
			minPerDay:  3334, // ceil(10000 / 3)
			minReached: false,
			minOnTrack: true,
			minBehind:  0,
		},
		{
			name:       "behind schedule raises per-day",
			goal:       mkGoal(CurrencyRUB, CurrencyRUB, utc(2024, 1, 1, 0), utc(2024, 1, 11, 0), 10000, 20000, 30000),
			now:        utc(2024, 1, 6, 0), // 5 of 10 days gone, nothing saved
			saved:      0,
			savedToday: 0,
			reached:    TierNone,
			daysLeft:   5,
			monthsLeft: 1,
			minRemain:  10000,
			minPerDay:  2000, // ceil(10000 / 5): double original 1000
			minReached: false,
			minOnTrack: false,
			minBehind:  5000, // half the target should have been saved by now
		},
		{
			name: "saved today is separate from saved total",
			goal: mkGoal(CurrencyRUB, CurrencyRUB, utc(2024, 1, 1, 0), utc(2024, 1, 11, 0), 10000, 20000, 30000),
			deposits: []Deposit{
				{Amount: 3000, HappenedAt: utc(2024, 1, 6, 0)}, // today
				{Amount: 2000, HappenedAt: utc(2024, 1, 2, 0)}, // earlier
			},
			now:        utc(2024, 1, 6, 0),
			saved:      5000,
			savedToday: 3000,
			reached:    TierNone,
			daysLeft:   5,
			monthsLeft: 1,
			minRemain:  5000,
			minPerDay:  1000,
			minReached: false,
			minOnTrack: true,
			minBehind:  0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st := ComputeStatus(c.goal, c.deposits, c.rate, c.now)

			if st.Saved != c.saved {
				t.Errorf("Saved = %d, want %d", st.Saved, c.saved)
			}
			if st.SavedToday != c.savedToday {
				t.Errorf("SavedToday = %d, want %d", st.SavedToday, c.savedToday)
			}
			if st.ReachedTier != c.reached {
				t.Errorf("ReachedTier = %v, want %v", st.ReachedTier, c.reached)
			}
			if st.DaysLeft != c.daysLeft {
				t.Errorf("DaysLeft = %d, want %d", st.DaysLeft, c.daysLeft)
			}
			if st.MonthsLeft != c.monthsLeft {
				t.Errorf("MonthsLeft = %d, want %d", st.MonthsLeft, c.monthsLeft)
			}

			min := st.Tiers[TierMinimal]
			if min.Remaining != c.minRemain {
				t.Errorf("min.Remaining = %d, want %d", min.Remaining, c.minRemain)
			}
			if min.PerDay != c.minPerDay {
				t.Errorf("min.PerDay = %d, want %d", min.PerDay, c.minPerDay)
			}
			if min.Reached != c.minReached {
				t.Errorf("min.Reached = %v, want %v", min.Reached, c.minReached)
			}
			if min.OnTrack != c.minOnTrack {
				t.Errorf("min.OnTrack = %v, want %v", min.OnTrack, c.minOnTrack)
			}
			if min.Behind != c.minBehind {
				t.Errorf("min.Behind = %d, want %d", min.Behind, c.minBehind)
			}
		})
	}
}
