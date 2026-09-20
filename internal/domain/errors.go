package domain

import "errors"

// ErrNotFound: returned by repositories when the record does not exist. Compare it
// with errors.Is.
var ErrNotFound = errors.New("not found")

// ErrInvalidGoal: returned by ValidateGoal when a goal fails the checks. Each
// wrapped error holds the reason; compare with errors.Is.
var ErrInvalidGoal = errors.New("invalid goal")
