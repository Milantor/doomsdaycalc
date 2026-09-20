package bot

import (
	"fmt"
	"strconv"
	"strings"

	"lab042.ru/doomsdaycalc/internal/domain"
	"lab042.ru/doomsdaycalc/internal/i18n"
)

// statusMark and checkMark: fixed symbols in the status text. Not language strings, so
// they stay here.
const (
	statusMark = "🎯"
	checkMark  = "✅"
)

// formatMoney: minor units as whole units of the currency, e.g. 3000 kopecks -> "30".
// Money is never float, so the split keeps integer math.
func formatMoney(m domain.Money, c domain.Currency) string {
	div := int64(1)
	for i := 0; i < c.Exponent(); i++ {
		div *= 10
	}
	return strconv.FormatInt(int64(m)/div, 10)
}

// formatStatus: renders a goal status into one message. Every tier reached switches to
// formatGoalDone; otherwise all three tiers share a line in minimal/acceptable/optimal
// order, and a reached target shows a check.
func formatStatus(g domain.Goal, st domain.Status, m i18n.Messages) string {
	if st.ReachedTier == domain.TierOptimal {
		return formatGoalDone(g, st, m)
	}

	money := func(v domain.Money) string { return formatMoney(v, g.Currency) }
	tier := func(f func(domain.TierStatus) string) string {
		parts := make([]string, 0, len(domain.Tiers))
		for _, t := range domain.Tiers {
			parts = append(parts, f(st.Tiers[t]))
		}
		return strings.Join(parts, "/")
	}
	cur := string(g.Currency)

	// A reached target becomes a check, an open one keeps its number.
	targets := tier(func(ts domain.TierStatus) string {
		if ts.Reached {
			return checkMark
		}
		return money(ts.Target)
	})

	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", statusMark, g.Title)
	fmt.Fprintf(&b, "%s: %s %s %s %s (%s)\n", m.StatusSavedLabel,
		money(st.Saved), m.StatusSavedOf, targets, cur, m.StatusTiers)
	fmt.Fprintf(&b, "%s: %s %s\n", m.StatusLeftLabel,
		tier(func(ts domain.TierStatus) string { return money(ts.Remaining) }), cur)
	fmt.Fprintf(&b, "%s: %s %s\n", m.StatusPerDayLabel,
		tier(func(ts domain.TierStatus) string { return money(ts.PerDay) }), cur)
	fmt.Fprintf(&b, "%s: %s %s\n", m.StatusPerMonthLabel,
		tier(func(ts domain.TierStatus) string { return money(ts.PerMonth) }), cur)
	fmt.Fprintf(&b, "%s: %s · %s\n", m.StatusDeadlineLabel,
		g.Deadline.Format("2006-01-02"), fmt.Sprintf(m.StatusDaysLeft, st.DaysLeft))
	pace := tier(func(ts domain.TierStatus) string {
		if ts.OnTrack {
			return checkMark
		}
		return money(ts.Behind)
	})
	// All on pace means the line is checks only, so the currency stays off.
	anyBehind := false
	for _, t := range domain.Tiers {
		if !st.Tiers[t].OnTrack {
			anyBehind = true
		}
	}
	if anyBehind {
		fmt.Fprintf(&b, "%s: %s %s", m.StatusPaceLabel, pace, cur)
	} else {
		fmt.Fprintf(&b, "%s: %s", m.StatusPaceLabel, pace)
	}
	return b.String()
}

// formatGoalDone: status of a goal with every tier reached. Targets, pace and per-day
// figures drop; title, saved total, done line and deadline stay.
func formatGoalDone(g domain.Goal, st domain.Status, m i18n.Messages) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", statusMark, g.Title)
	fmt.Fprintf(&b, "%s %s\n", formatMoney(st.Saved, g.Currency), g.Currency)
	fmt.Fprintf(&b, "%s\n", m.GoalDone)
	fmt.Fprintf(&b, "%s: %s", m.StatusDeadlineLabel, g.Deadline.Format("2006-01-02"))
	return b.String()
}
