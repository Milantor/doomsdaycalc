package service

import "errors"

// ErrUnknownScenario: Begin was asked for a scenario name that is not registered.
var ErrUnknownScenario = errors.New("unknown scenario")

// ErrCancelled: a dialogue ended with nothing to store, e.g. a zero amount in a money
// dialogue. Answer returns it with running false and the position already dropped.
var ErrCancelled = errors.New("cancelled")

// ErrOverdraw: a withdrawal asked for more than the goal holds. Answer returns it with
// running true, so the amount question is asked again and the position stays.
var ErrOverdraw = errors.New("overdraw")

// ErrTierOrder: a target dropped below the tier before it. Answer returns it with
// running true, so the amount question is asked again and the position stays.
var ErrTierOrder = errors.New("tier order")
