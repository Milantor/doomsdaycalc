package domain

import (
	"math/big"
	"time"
)

// Progress calculation: pure, no clock/database/Telegram. `now` comes in as a
// parameter, which makes it testable without any I/O.

// ExchangeRate: converts an amount from Goal.SavingsCurrency into Goal.Currency.
// Integer math (m * Num / Den, rounded half up) keeps float away from money. Zero
// value means identity, so it is safe to leave unset.
type ExchangeRate struct {
	Num int64
	Den int64
}

// IdentityRate: the 1:1 rate, used when both currencies are the same.
func IdentityRate() ExchangeRate { return ExchangeRate{Num: 1, Den: 1} }

// Convert returns m in the target currency.
func (r ExchangeRate) Convert(m Money) Money {
	if r.Num == 0 || r.Den == 0 {
		return m
	}
	num, den := r.Num, r.Den
	if den < 0 {
		num, den = -num, -den
	}
	n := int64(m) * num
	q := n / den
	if rem := n % den; rem*2 >= den {
		q++
	}
	return Money(q)
}

// TierStatus: progress towards one of the three targets. Amounts are in
// Goal.Currency.
type TierStatus struct {
	Target    Money // tier target
	Remaining Money // Target - Saved, never negative
	Reached   bool  // Saved >= Target
	PerDay    Money // Remaining spread over DaysLeft, rounded up
	PerMonth  Money // Remaining spread over MonthsLeft, rounded up
	OnTrack   bool  // Saved >= steady pace expected by now
	Behind    Money // arrears vs steady pace: max(0, expected - Saved)
}

// Status: computed progress of a goal at one moment.
type Status struct {
	Saved       Money               // total deposited so far, in Goal.Currency
	SavedToday  Money               // deposited today (goals timezone), in Goal.Currency
	ReachedTier Tier                // highest tier fully reached, or TierNone
	DaysLeft    int                 // whole days until deadline, never negative
	MonthsLeft  int                 // whole calendar months left, at least 1 while ahead
	Tiers       map[Tier]TierStatus // one entry per tier
}

// ComputeStatus: the pure progress calculation. Never reads the clock, caller
// passes `now`. `rate` is used only when Goal.Currency and Goal.SavingsCurrency
// differ; when they match it is ignored.
func ComputeStatus(g Goal, deposits []Deposit, rate ExchangeRate, now time.Time) Status {
	if g.Currency == g.SavingsCurrency {
		rate = IdentityRate()
	}

	// What "today"/"this month" means depends on the user timezone. Unknown zone
	// falls back to UTC.
	loc := time.UTC
	if g.Timezone != "" {
		if l, err := time.LoadLocation(g.Timezone); err == nil {
			loc = l
		}
	}

	// Deposits are in SavingsCurrency, targets in Currency. Sum total and the part
	// made today, then convert both through the rate.
	startOfToday := startOfDay(now, loc)
	var deposited, today Money
	for _, d := range deposits {
		deposited += d.Amount
		if !d.HappenedAt.Before(startOfToday) {
			today += d.Amount
		}
	}

	// Pace counts whole days and the day in progress counts, so the installment for that
	// day shows at once. The anchor is the start of the creation day and the pace reference
	// the end of the current day; both keep the figures steady within the day.
	anchor := startOfDay(g.CreatedAt, loc)
	paceRef := startOfToday.AddDate(0, 0, 1)

	st := Status{
		Saved:       rate.Convert(deposited),
		SavedToday:  rate.Convert(today),
		ReachedTier: TierNone,
		DaysLeft:    daysUntil(now, g.Deadline),
		MonthsLeft:  monthsUntil(now, g.Deadline, loc),
		Tiers:       make(map[Tier]TierStatus, len(Tiers)),
	}
	for _, t := range Tiers {
		ts := tierStatus(st.Saved, g.Targets[t], anchor, g.Deadline, paceRef, st.DaysLeft, st.MonthsLeft)
		if ts.Reached {
			st.ReachedTier = t // Tiers is ascending, so this ends at the highest reached
		}
		st.Tiers[t] = ts
	}
	return st
}

// tierStatus: computes one tier; daysLeft and monthsLeft are shared by all. anchor is the
// start of the creation day and paceRef the end of the current day, the two points the pace
// counts between.
func tierStatus(saved, target Money, anchor, deadline, paceRef time.Time, daysLeft, monthsLeft int) TierStatus {
	ts := TierStatus{Target: target}
	if saved >= target {
		ts.Reached = true
	} else {
		ts.Remaining = target - saved
	}
	ts.PerDay = ceilDiv(ts.Remaining, Money(max(daysLeft, 1)))
	ts.PerMonth = ceilDiv(ts.Remaining, Money(max(monthsLeft, 1)))

	// Steady pace is what should already be saved by now. Falling below it means
	// arrears, which is why the next PerDay is higher.
	expected := steadyPace(target, anchor, deadline, paceRef)
	ts.OnTrack = saved >= expected
	if saved < expected {
		ts.Behind = expected - saved
	}
	return ts
}

// steadyPace: share of the target due by paceRef if money were set aside evenly from the
// anchor to Deadline. paceRef is the end of the current day, so a day the goal is held
// counts in full and the pace moves once a day. Empty window means the whole target is due;
// before the anchor nothing is due.
func steadyPace(target Money, anchor, deadline, paceRef time.Time) Money {
	total := deadline.Sub(anchor)
	if total <= 0 {
		return target
	}
	elapsed := paceRef.Sub(anchor)
	if elapsed <= 0 {
		return 0
	}
	if elapsed > total {
		elapsed = total
	}
	// A nanosecond elapsed time times a large target passes int64 max, so the product
	// goes through big.Int. Division truncates, as before.
	num := new(big.Int).Mul(big.NewInt(int64(target)), big.NewInt(int64(elapsed)))
	num.Quo(num, big.NewInt(int64(total)))
	return Money(num.Int64())
}

// startOfDay: midnight of the calendar day of now, in loc.
func startOfDay(now time.Time, loc *time.Location) time.Time {
	n := now.In(loc)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, loc)
}

// daysUntil: whole days from now to deadline, rounded up (a partial day counts),
// never negative.
func daysUntil(now, deadline time.Time) int {
	d := deadline.Sub(now)
	if d <= 0 {
		return 0
	}
	const day = 24 * time.Hour
	days := int(d / day)
	if d%day != 0 {
		days++
	}
	return days
}

// monthsUntil: whole calendar months between now and deadline in loc, floored, at
// least 1 while the deadline is ahead. Calendar months, since "save X per month" is
// a calendar idea.
func monthsUntil(now, deadline time.Time, loc *time.Location) int {
	if !deadline.After(now) {
		return 0
	}
	n := now.In(loc)
	d := deadline.In(loc)
	months := (d.Year()-n.Year())*12 + int(d.Month()-n.Month())
	if d.Day() < n.Day() {
		months--
	}
	if months < 1 {
		months = 1
	}
	return months
}

// ceilDiv divides a by b rounding up; zero when a <= 0 or b <= 0.
func ceilDiv(a, b Money) Money {
	if a <= 0 || b <= 0 {
		return 0
	}
	return (a + b - 1) / b
}
