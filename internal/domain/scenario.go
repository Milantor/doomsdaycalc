package domain

import (
	"context"
	"time"
)

// ScenarioState: where a user sits inside a dialogue. Stored in
// user_scenario_state, so a bot restart does not lose the position. Vars holds the
// answers collected so far, all as text; the caller parses them when the dialogue
// ends.
type ScenarioState struct {
	UserID       int64
	ScenarioName string
	NodeID       string
	EnteredAt    time.Time
	Vars         map[string]string
}

// ScenarioStateRepository: stores dialogue positions. Implemented in
// storage/postgres. Get returns ErrNotFound when the user has no dialogue running.
type ScenarioStateRepository interface {
	Get(ctx context.Context, userID int64) (ScenarioState, error)
	Save(ctx context.Context, s ScenarioState) error
	Delete(ctx context.Context, userID int64) error
}
