package domain

// Motivation after a money move. Only the bucket is decided here: the phrases are in
// i18n and the pick is random, so this file stays pure and testable.

// Mood: how big a money move is. A deposit is measured against fixed amounts, a
// withdrawal against the saved total it comes out of. One mood maps to one pool of
// phrases in the catalog.
type Mood int

const (
	MoodNone Mood = iota // no money moved, so no phrase of its own
	MoodDepositSmall
	MoodDepositMid
	MoodDepositBig
	MoodWithdrawSmall
	MoodWithdrawMid
	MoodWithdrawBig
)

// Moods: every mood that carries phrases, deposits first. MoodNone stays out of the
// list: it means the plain done reply.
var Moods = []Mood{
	MoodDepositSmall, MoodDepositMid, MoodDepositBig,
	MoodWithdrawSmall, MoodWithdrawMid, MoodWithdrawBig,
}

// Withdrawal switch points, in whole percent of the saved total. Below moodMidShare is
// small, at or above moodBigShare is big.
const (
	moodMidShare = 10
	moodBigShare = 25
)

// temp shit
// Deposit switch points, in whole units. 5000 or more reads mid, 15000 or more reads big.
// Scaled by 100 here, the same scale the service applies to a typed amount.
const (
	moodDepositMid = 5000 * 100
	moodDepositBig = 15000 * 100
)

// String: mood label, for logs and test messages.
func (m Mood) String() string {
	switch m {
	case MoodNone:
		return "none"
	case MoodDepositSmall:
		return "deposit_small"
	case MoodDepositMid:
		return "deposit_mid"
	case MoodDepositBig:
		return "deposit_big"
	case MoodWithdrawSmall:
		return "withdraw_small"
	case MoodWithdrawMid:
		return "withdraw_mid"
	case MoodWithdrawBig:
		return "withdraw_big"
	default:
		return "unknown"
	}
}

// DepositMood: mood of a deposit of amount in minor units, by fixed amounts, so the goal
// size is not used.
func DepositMood(amount Money) Mood {
	switch {
	case amount >= moodDepositBig:
		return MoodDepositBig
	case amount >= moodDepositMid:
		return MoodDepositMid
	default:
		return MoodDepositSmall
	}
}

// WithdrawMood: mood of a withdrawal, amount being the positive size of the move and
// saved the total before it. The share is how much of the savings is gone, so the goal
// target is not used.
func WithdrawMood(amount, saved Money) Mood {
	share := sharePercent(amount, saved)
	switch {
	case share >= moodBigShare:
		return MoodWithdrawBig
	case share >= moodMidShare:
		return MoodWithdrawMid
	default:
		return MoodWithdrawSmall
	}
}

// sharePercent: part as a whole percent of whole, truncated. A non-positive whole gives
// zero, which keeps an empty pile at the small mood and keeps the division safe.
func sharePercent(part, whole Money) int {
	if whole <= 0 {
		return 0
	}
	return int(part * 100 / whole)
}
