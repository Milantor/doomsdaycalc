package domain

import (
	"fmt"
	"strings"
	"time"
)

// ValidateGoal: checks a goal before storing it. Called at service boundary, since
// a bad target gives wrong numbers in every later status calculation. `now` comes in
// as a parameter, so the deadline rule is testable. Each failure wraps
// ErrInvalidGoal; compare with errors.Is.
func ValidateGoal(g Goal, now time.Time) error {
	if strings.TrimSpace(g.Title) == "" {
		return fmt.Errorf("%w: title is empty", ErrInvalidGoal)
	}
	if !g.Deadline.After(now) {
		return fmt.Errorf("%w: deadline is not in the future", ErrInvalidGoal)
	}
	if !g.Currency.Valid() {
		return fmt.Errorf("%w: bad currency %q", ErrInvalidGoal, g.Currency)
	}
	if !g.SavingsCurrency.Valid() {
		return fmt.Errorf("%w: bad savings currency %q", ErrInvalidGoal, g.SavingsCurrency)
	}

	// Every target is required and positive: a zero target makes saved >= target
	// always true, so that tier and every higher one look reached.
	tMin, tOK, tMax := g.Targets[TierMinimal], g.Targets[TierAcceptable], g.Targets[TierOptimal]
	if tMin <= 0 || tOK <= 0 || tMax <= 0 {
		return fmt.Errorf("%w: every target must be positive", ErrInvalidGoal)
	}
	if tMin > tOK || tOK > tMax {
		return fmt.Errorf("%w: targets ascend minimal <= acceptable <= optimal", ErrInvalidGoal)
	}
	return nil
}
