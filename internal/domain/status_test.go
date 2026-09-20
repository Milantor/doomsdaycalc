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
		{"negative denominator normalizes", 1000, ExchangeRate{Num: -1, Den: -2}, 500},
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
			minOnTrack: false,
			minBehind:  1000, // one day of the ten day window
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
			minOnTrack: false,
			minBehind:  3333, // 10000 * 1 / 3
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
			minBehind:  6000, // 10000 * 6 / 10, the day in progress counts
		},
		{
			name:       "creation day owes the first day",
			goal:       mkGoal(CurrencyRUB, CurrencyRUB, utc(2024, 1, 1, 10), utc(2024, 1, 11, 0), 10000, 20000, 30000),
			now:        utc(2024, 1, 1, 10), // the day in progress counts, so day one is already owed
			saved:      0,
			savedToday: 0,
			reached:    TierNone,
			daysLeft:   10,
			monthsLeft: 1,
			minRemain:  10000,
			minPerDay:  1000,
			minReached: false,
			minOnTrack: false,
			minBehind:  1000, // 10000 * 1 / 10
		},
		{
			name:       "pace counts the day in progress",
			goal:       mkGoal(CurrencyRUB, CurrencyRUB, utc(2024, 1, 1, 15), utc(2024, 1, 11, 0), 10000, 20000, 30000),
			now:        utc(2024, 1, 3, 9), // third day of a 10 day window
			saved:      0,
			savedToday: 0,
			reached:    TierNone,
			daysLeft:   8,
			monthsLeft: 1,
			minRemain:  10000,
			minPerDay:  1250, // ceil(10000 / 8)
			minReached: false,
			minOnTrack: false,
			minBehind:  3000, // 10000 * 3 / 10
		},
		{
			name:       "large targets keep the pace exact",
			goal:       mkGoal(CurrencyRUB, CurrencyRUB, utc(2024, 1, 1, 0), utc(2024, 1, 11, 0), 100000, 200000, 300000),
			now:        utc(2024, 1, 6, 0), // 5 of 10 days gone
			saved:      0,
			savedToday: 0,
			reached:    TierNone,
			daysLeft:   5,
			monthsLeft: 1,
			minRemain:  100000,
			minPerDay:  20000,
			minReached: false,
			minOnTrack: false,
			minBehind:  60000, // 100000 * 6 / 10
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
			minOnTrack: false,
			minBehind:  1000, // 10000 * 6 / 10 minus 5000 saved
		},
		{
			name:       "deadline on the creation day owes the whole target",
			goal:       mkGoal(CurrencyRUB, CurrencyRUB, utc(2024, 1, 5, 10), utc(2024, 1, 5, 0), 10000, 20000, 30000),
			now:        utc(2024, 1, 5, 10),
			saved:      0,
			savedToday: 0,
			reached:    TierNone,
			daysLeft:   0,
			monthsLeft: 0,
			minRemain:  10000,
			minPerDay:  10000, // the only day there is
			minReached: false,
			minOnTrack: false,
			minBehind:  10000,
		},
		{
			name:       "highest reached tier is optimal",
			goal:       mkGoal(CurrencyRUB, CurrencyRUB, utc(2024, 1, 1, 0), utc(2024, 1, 11, 0), 10000, 20000, 30000),
			deposits:   []Deposit{{Amount: 30000, HappenedAt: utc(2024, 1, 6, 0)}},
			now:        utc(2024, 1, 6, 0),
			saved:      30000,
			savedToday: 30000,
			reached:    TierOptimal,
			daysLeft:   5,
			monthsLeft: 1,
			minRemain:  0,
			minPerDay:  0,
			minReached: true,
			minOnTrack: true,
			minBehind:  0,
		},
		{
			name:       "goal created after now holds no debt",
			goal:       mkGoal(CurrencyRUB, CurrencyRUB, utc(2024, 1, 5, 10), utc(2024, 1, 11, 0), 10000, 20000, 30000),
			now:        utc(2024, 1, 1, 10), // before the creation day, so the pace reference is not reached
			saved:      0,
			savedToday: 0,
			reached:    TierNone,
			daysLeft:   10,
			monthsLeft: 1,
			minRemain:  10000,
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

// TestSteadyPaceAllTiers: the three tiers share one pace shape, so a higher target never
// sits less behind than a lower one. With no deposits the numbers are each target share of
// the days gone.
func TestSteadyPaceAllTiers(t *testing.T) {
	base := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	g := mkGoal(CurrencyRUB, CurrencyRUB, base.Add(9*time.Hour), base.AddDate(0, 0, 1024), 15000000, 25000000, 30000000)
	now := base.AddDate(0, 0, 255) // 256 days count with the day in progress, a quarter of the window

	st := ComputeStatus(g, nil, IdentityRate(), now)

	cases := []struct {
		tier   Tier
		behind Money
	}{
		{TierMinimal, 3750000},    // 15000000 / 4
		{TierAcceptable, 6250000}, // 25000000 / 4
		{TierOptimal, 7500000},    // 30000000 / 4
	}
	for _, c := range cases {
		ts := st.Tiers[c.tier]
		if ts.OnTrack {
			t.Errorf("%v is on track, want behind", c.tier)
		}
		if ts.Behind != c.behind {
			t.Errorf("%v behind = %d, want %d", c.tier, ts.Behind, c.behind)
		}
	}
}

// TestSteadyPaceNoOverflow: a long window, large targets and a deadline with a sub-day
// remainder. The bare product target*elapsed passes int64 max here, so a plain multiply
// wraps and breaks the tier order.
func TestSteadyPaceNoOverflow(t *testing.T) {
	g := mkGoal(CurrencyRUB, CurrencyRUB, utc(2000, 1, 1, 9), utc(2010, 1, 1, 0), 15000000, 25000000, 30000000)
	g.Deadline = g.Deadline.Add(time.Millisecond) // sub-day remainder
	now := time.Date(2005, 1, 1, 0, 0, 0, 0, time.UTC)

	st := ComputeStatus(g, nil, IdentityRate(), now)
	min := st.Tiers[TierMinimal]
	ok := st.Tiers[TierAcceptable]
	opt := st.Tiers[TierOptimal]
	if min.OnTrack || ok.OnTrack || opt.OnTrack {
		t.Fatalf("tiers on track with nothing saved: %v/%v/%v", min.OnTrack, ok.OnTrack, opt.OnTrack)
	}
	if !(min.Behind < ok.Behind && ok.Behind < opt.Behind) {
		t.Fatalf("behind not ordered: %d/%d/%d", min.Behind, ok.Behind, opt.Behind)
	}
}

// TestPaceStableWithinDay: the pace moves once a day. Every moment of the same day gives
// the same figures, so a second look does not shift them.
func TestPaceStableWithinDay(t *testing.T) {
	g := mkGoal(CurrencyRUB, CurrencyRUB, utc(2024, 1, 1, 9), utc(2024, 1, 11, 0), 15000000, 25000000, 30000000)
	day := time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)
	base := ComputeStatus(g, nil, IdentityRate(), day)
	for _, h := range []int{1, 8, 13, 23} {
		now := time.Date(2024, 1, 5, h, 30, 0, 123456789, time.UTC)
		st := ComputeStatus(g, nil, IdentityRate(), now)
		for _, tier := range Tiers {
			if st.Tiers[tier].Behind != base.Tiers[tier].Behind {
				t.Errorf("hour %d: %v behind = %d, want %d", h, tier, st.Tiers[tier].Behind, base.Tiers[tier].Behind)
			}
		}
	}
	next := ComputeStatus(g, nil, IdentityRate(), time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC))
	if next.Tiers[TierMinimal].Behind == base.Tiers[TierMinimal].Behind {
		t.Errorf("next day keeps pace %d, want a move", base.Tiers[TierMinimal].Behind)
	}
}

// TestSteadyPaceMonotonic: with nothing saved, behind grows with the target at every
// moment of the window.
func TestSteadyPaceMonotonic(t *testing.T) {
	g := mkGoal(CurrencyRUB, CurrencyRUB, utc(2024, 1, 1, 10), utc(2024, 1, 11, 0), 15000000, 25000000, 30000000)
	for d := 0; d <= 10; d++ {
		for _, h := range []int{0, 7, 13, 23} {
			now := time.Date(2024, 1, 1+d, h, 0, 0, 123456789, time.UTC)
			st := ComputeStatus(g, nil, IdentityRate(), now)
			min := st.Tiers[TierMinimal]
			ok := st.Tiers[TierAcceptable]
			opt := st.Tiers[TierOptimal]
			if min.Behind > ok.Behind || ok.Behind > opt.Behind {
				t.Errorf("now %s: behind not monotonic: %d/%d/%d", now, min.Behind, ok.Behind, opt.Behind)
			}
		}
	}
}

// TestPaceOnTrackBoundary: saved equal to the expected pace is on track, one minor unit
// under is behind by one.
func TestPaceOnTrackBoundary(t *testing.T) {
	g := mkGoal(CurrencyRUB, CurrencyRUB, utc(2024, 1, 1, 9), utc(2024, 1, 11, 0), 10000, 20000, 30000)
	now := utc(2024, 1, 6, 3) // six of ten days counted, expected 6000

	at := ComputeStatus(g, []Deposit{{Amount: 6000, HappenedAt: now}}, IdentityRate(), now)
	if !at.Tiers[TierMinimal].OnTrack || at.Tiers[TierMinimal].Behind != 0 {
		t.Errorf("saved 6000: onTrack=%v behind=%d, want true/0", at.Tiers[TierMinimal].OnTrack, at.Tiers[TierMinimal].Behind)
	}
	under := ComputeStatus(g, []Deposit{{Amount: 5999, HappenedAt: now}}, IdentityRate(), now)
	if under.Tiers[TierMinimal].OnTrack || under.Tiers[TierMinimal].Behind != 1 {
		t.Errorf("saved 5999: onTrack=%v behind=%d, want false/1", under.Tiers[TierMinimal].OnTrack, under.Tiers[TierMinimal].Behind)
	}
}

// TestComputeStatusTimezone: "today" follows the goal timezone, so one deposit lands in
// today or in yesterday depending on the zone.
func TestComputeStatusTimezone(t *testing.T) {
	now := time.Date(2024, 1, 6, 0, 30, 0, 0, time.UTC)
	dep := []Deposit{{Amount: 50000, HappenedAt: time.Date(2024, 1, 5, 23, 30, 0, 0, time.UTC)}}
	g := mkGoal(CurrencyRUB, CurrencyRUB, utc(2024, 1, 1, 0), utc(2024, 1, 11, 0), 100000, 200000, 300000)

	g.Timezone = "UTC"
	if got := ComputeStatus(g, dep, IdentityRate(), now).SavedToday; got != 0 {
		t.Errorf("UTC savedToday = %d, want 0", got)
	}

	g.Timezone = "Europe/Berlin" // +01:00 in winter, so 23:30 UTC is already the next day
	if got := ComputeStatus(g, dep, IdentityRate(), now).SavedToday; got != 50000 {
		t.Errorf("Berlin savedToday = %d, want 50000", got)
	}
}

// TestComputeStatusUnknownTimezone: an unknown zone falls back to UTC.
func TestComputeStatusUnknownTimezone(t *testing.T) {
	now := time.Date(2024, 1, 6, 0, 30, 0, 0, time.UTC)
	dep := []Deposit{{Amount: 50000, HappenedAt: time.Date(2024, 1, 5, 23, 30, 0, 0, time.UTC)}}
	g := mkGoal(CurrencyRUB, CurrencyRUB, utc(2024, 1, 1, 0), utc(2024, 1, 11, 0), 100000, 200000, 300000)
	g.Timezone = "Nope/Nowhere"

	if got := ComputeStatus(g, dep, IdentityRate(), now).SavedToday; got != 0 {
		t.Errorf("unknown zone savedToday = %d, want 0", got)
	}
}

// TestComputeStatusSavedTodayBoundary: a deposit at the start of today counts as today, one
// nanosecond earlier does not.
func TestComputeStatusSavedTodayBoundary(t *testing.T) {
	midnight := time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC)
	now := time.Date(2024, 1, 6, 9, 0, 0, 0, time.UTC)
	g := mkGoal(CurrencyRUB, CurrencyRUB, utc(2024, 1, 1, 0), utc(2024, 1, 11, 0), 100000, 200000, 300000)

	at := ComputeStatus(g, []Deposit{{Amount: 1, HappenedAt: midnight}}, IdentityRate(), now)
	if at.SavedToday != 1 {
		t.Errorf("deposit at midnight savedToday = %d, want 1", at.SavedToday)
	}
	before := ComputeStatus(g, []Deposit{{Amount: 1, HappenedAt: midnight.Add(-time.Nanosecond)}}, IdentityRate(), now)
	if before.SavedToday != 0 {
		t.Errorf("deposit a nanosecond earlier savedToday = %d, want 0", before.SavedToday)
	}
}
