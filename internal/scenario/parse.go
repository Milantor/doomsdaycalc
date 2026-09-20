package scenario

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Node.Parse helpers. They only reshape text: no domain types, no I/O, so the
// scenario package stays pure. Node.Parse returns ErrBadAnswer through Step when one
// of these refuses the input.

// parseTitle: trimmed, and it must not be empty.
func parseTitle(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "", errors.New("title is empty")
	}
	return v, nil
}

// parseDate: a date in YYYY-MM-DD, rewritten to the canonical form.
func parseDate(raw string) (string, error) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("date %q is not YYYY-MM-DD", raw)
	}
	return t.Format("2006-01-02"), nil
}

// parseAmount: a positive whole number, rewritten without stray spaces.
func parseAmount(raw string) (string, error) {
	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return "", fmt.Errorf("amount %q is not a whole number", raw)
	}
	if n <= 0 {
		return "", fmt.Errorf("amount %q is not positive", raw)
	}
	return strconv.FormatInt(n, 10), nil
}

// parseMoneyAmount: a whole number for a money dialogue, rewritten without stray
// spaces. Zero is allowed and cancels the dialogue; the service then stores nothing.
func parseMoneyAmount(raw string) (string, error) {
	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return "", fmt.Errorf("amount %q is not a whole number", raw)
	}
	if n < 0 {
		return "", fmt.Errorf("amount %q is negative", raw)
	}
	return strconv.FormatInt(n, 10), nil
}
