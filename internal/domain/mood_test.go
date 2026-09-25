package domain

import (
	"slices"
	"testing"
)

// TestMoodNoneIsZero: MoodNone is the zero value, so an unset mood means no money moved.
func TestMoodNoneIsZero(t *testing.T) {
	var m Mood
	if m != MoodNone {
		t.Fatalf("zero Mood = %s, want %s", m, MoodNone)
	}
}

// TestMoodString: every mood has a label, a value outside the set falls back.
func TestMoodString(t *testing.T) {
	cases := []struct {
		name string
		mood Mood
		want string
	}{
		{"none", MoodNone, "none"},
		{"deposit small", MoodDepositSmall, "deposit_small"},
		{"deposit mid", MoodDepositMid, "deposit_mid"},
		{"deposit big", MoodDepositBig, "deposit_big"},
		{"withdraw small", MoodWithdrawSmall, "withdraw_small"},
		{"withdraw mid", MoodWithdrawMid, "withdraw_mid"},
		{"withdraw big", MoodWithdrawBig, "withdraw_big"},
		{"one above the set", MoodWithdrawBig + 1, "unknown"},
		{"far above the set", Mood(99), "unknown"},
		{"negative", Mood(-1), "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.mood.String(); got != c.want {
				t.Fatalf("Mood(%d).String() = %q, want %q", c.mood, got, c.want)
			}
		})
	}
}

// TestMoods: the list holds every mood that carries phrases, in order, once each, and
// leaves MoodNone out.
func TestMoods(t *testing.T) {
	want := []Mood{
		MoodDepositSmall, MoodDepositMid, MoodDepositBig,
		MoodWithdrawSmall, MoodWithdrawMid, MoodWithdrawBig,
	}
	if !slices.Equal(Moods, want) {
		t.Fatalf("Moods = %v, want %v", Moods, want)
	}
	if slices.Contains(Moods, MoodNone) {
		t.Error("Moods holds MoodNone")
	}
}

// TestMoodsHaveLabels: every mood in the list has a label of its own, so a mood added to
// the list without a String case shows up here.
func TestMoodsHaveLabels(t *testing.T) {
	for _, m := range Moods {
		if got := m.String(); got == "unknown" {
			t.Errorf("Mood(%d) has no label", m)
		}
	}
}

// TestDepositMood: the mood is set by fixed amounts. Switch points are 5000 and 15000
// whole units, so the cases sit on and next to both.
func TestDepositMood(t *testing.T) {
	cases := []struct {
		name   string
		amount Money
		want   Mood
	}{
		{"nothing deposited", 0, MoodDepositSmall},
		{"one kopeck", 1, MoodDepositSmall},
		{"just below mid", 4999 * 100, MoodDepositSmall},
		{"at mid", 5000 * 100, MoodDepositMid},
		{"one kopeck above mid", 5000*100 + 1, MoodDepositMid},
		{"just below big", 14999 * 100, MoodDepositMid},
		{"at big", 15000 * 100, MoodDepositBig},
		{"one kopeck above big", 15000*100 + 1, MoodDepositBig},
		{"far over big", 1000000 * 100, MoodDepositBig},
		{"negative amount", -5000 * 100, MoodDepositSmall},
		{"huge amount", 25_000_000_000, MoodDepositBig},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := DepositMood(c.amount); got != c.want {
				t.Fatalf("DepositMood(%d) = %s, want %s", c.amount, got, c.want)
			}
		})
	}
}

// TestWithdrawMood: the mood is set by the share of the saved total, so the goal target
// is not used. Switch points are 10 and 25 percent.
func TestWithdrawMood(t *testing.T) {
	cases := []struct {
		name   string
		amount Money
		saved  Money
		want   Mood
	}{
		{"nothing withdrawn", 0, 50000, MoodWithdrawSmall},
		{"one kopeck out", 1, 50000, MoodWithdrawSmall},
		{"just below mid", 4999, 50000, MoodWithdrawSmall},
		{"at mid", 5000, 50000, MoodWithdrawMid},
		{"one above mid", 5001, 50000, MoodWithdrawMid},
		{"just below big", 12499, 50000, MoodWithdrawMid},
		{"at big", 12500, 50000, MoodWithdrawBig},
		{"one above big", 12501, 50000, MoodWithdrawBig},
		{"everything saved", 50000, 50000, MoodWithdrawBig},
		{"over the saved total", 60000, 50000, MoodWithdrawBig},
		{"nothing saved", 1, 0, MoodWithdrawSmall},
		{"negative saved total", 1, -10000, MoodWithdrawSmall},
		{"negative amount", -5000, 50000, MoodWithdrawSmall},
		{"huge amounts", 25_000_000_000, 100_000_000_000, MoodWithdrawBig},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := WithdrawMood(c.amount, c.saved); got != c.want {
				t.Fatalf("WithdrawMood(%d, %d) = %s, want %s", c.amount, c.saved, got, c.want)
			}
		})
	}
}

// TestDepositMoodEveryWholeUnit: walks every whole unit up to 20000, so both switch
// points are checked exactly.
func TestDepositMoodEveryWholeUnit(t *testing.T) {
	for unit := 0; unit <= 20000; unit++ {
		want := MoodDepositBig
		switch {
		case unit < 5000:
			want = MoodDepositSmall
		case unit < 15000:
			want = MoodDepositMid
		}

		amount := Money(unit) * 100
		if got := DepositMood(amount); got != want {
			t.Fatalf("%d whole units: DepositMood(%d) = %s, want %s", unit, amount, got, want)
		}
	}
}

// TestWithdrawMoodEveryPercent: same walk for a withdrawal, where the denominator is the
// saved total.
func TestWithdrawMoodEveryPercent(t *testing.T) {
	const saved Money = 10000
	for share := 0; share <= 150; share++ {
		amount := Money(share) * saved / 100
		want := MoodWithdrawBig
		switch {
		case share < 10:
			want = MoodWithdrawSmall
		case share < 25:
			want = MoodWithdrawMid
		}
		if got := WithdrawMood(amount, saved); got != want {
			t.Fatalf("%d%%: WithdrawMood(%d, %d) = %s, want %s", share, amount, saved, got, want)
		}
	}
}

// TestMoodsDontMix: a deposit never reports a withdrawal mood and a withdrawal never
// reports a deposit mood, which the two separate sets guard.
func TestMoodsDontMix(t *testing.T) {
	deposits := []Mood{MoodDepositSmall, MoodDepositMid, MoodDepositBig}
	withdrawals := []Mood{MoodWithdrawSmall, MoodWithdrawMid, MoodWithdrawBig}

	// 20000 whole units in steps, against a pile of the same size.
	const pile Money = 20000 * 100
	for unit := 0; unit <= 20000; unit += 250 {
		amount := Money(unit) * 100
		if got := DepositMood(amount); !slices.Contains(deposits, got) {
			t.Fatalf("%d whole units: deposit mood = %s, not a deposit mood", unit, got)
		}
		if got := WithdrawMood(amount, pile); !slices.Contains(withdrawals, got) {
			t.Fatalf("%d whole units: withdrawal mood = %s, not a withdrawal mood", unit, got)
		}
	}
}

// TestSharePercent: the percent is truncated towards zero, a non-positive whole gives
// zero, and a part over the whole goes past 100.
func TestSharePercent(t *testing.T) {
	cases := []struct {
		name        string
		part, whole Money
		want        int
	}{
		{"nothing of something", 0, 100, 0},
		{"one percent", 1, 100, 1},
		{"half", 50, 100, 50},
		{"ninety nine percent", 99, 100, 99},
		{"the whole", 100, 100, 100},
		{"more than the whole", 250, 100, 250},
		{"truncates down", 199, 1000, 19},
		{"truncates just under a point", 999, 1000, 99},
		{"a single minor unit from a single minor unit", 1, 1, 100},
		{"nothing from a single minor unit", 0, 1, 0},
		{"part of nothing", 10, 0, 0},
		{"negative whole", 10, -5, 0},
		{"negative part", -10, 100, -10},
		{"huge amounts", 25_000_000_000, 100_000_000_000, 25},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := sharePercent(c.part, c.whole); got != c.want {
				t.Fatalf("sharePercent(%d, %d) = %d, want %d", c.part, c.whole, got, c.want)
			}
		})
	}
}
