package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lab042.ru/doomsdaycalc/internal/domain"
)

// ScenarioRepo: PostgreSQL implementation of domain.ScenarioStateRepository.
type ScenarioRepo struct {
	pool *pgxpool.Pool
}

// NewScenarioRepo: scenario state repo backed by the given pool.
func NewScenarioRepo(pool *pgxpool.Pool) *ScenarioRepo {
	return &ScenarioRepo{pool: pool}
}

// ScenarioRepo implements domain.ScenarioStateRepository.
var _ domain.ScenarioStateRepository = (*ScenarioRepo)(nil)

// Get returns the dialogue position of the user, or domain.ErrNotFound.
func (r *ScenarioRepo) Get(ctx context.Context, userID int64) (domain.ScenarioState, error) {
	const q = `
SELECT user_id, scenario_name, node_id, entered_at, vars
FROM user_scenario_state
WHERE user_id = $1`

	var s domain.ScenarioState
	err := r.pool.QueryRow(ctx, q, userID).
		Scan(&s.UserID, &s.ScenarioName, &s.NodeID, &s.EnteredAt, &s.Vars)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ScenarioState{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.ScenarioState{}, fmt.Errorf("get scenario state of user %d: %w", userID, err)
	}
	return s, nil
}

// Save upserts the dialogue position. One row per user: starting or advancing a
// dialogue replaces the old position. Zero EnteredAt becomes the current time.
func (r *ScenarioRepo) Save(ctx context.Context, s domain.ScenarioState) error {
	if s.EnteredAt.IsZero() {
		s.EnteredAt = time.Now().UTC()
	}
	// jsonb column: null would come back as a nil map, so store an empty object.
	vars := s.Vars
	if vars == nil {
		vars = map[string]string{}
	}

	const q = `
INSERT INTO user_scenario_state (user_id, scenario_name, node_id, entered_at, vars)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id) DO UPDATE
SET scenario_name = EXCLUDED.scenario_name,
    node_id       = EXCLUDED.node_id,
    entered_at    = EXCLUDED.entered_at,
    vars          = EXCLUDED.vars`

	if _, err := r.pool.Exec(ctx, q, s.UserID, s.ScenarioName, s.NodeID, s.EnteredAt, vars); err != nil {
		return fmt.Errorf("save scenario state of user %d: %w", s.UserID, err)
	}
	return nil
}

// Delete drops the dialogue position. A missing row is not an error: every dialogue
// ends by deleting, and a repeated end must stay quiet.
func (r *ScenarioRepo) Delete(ctx context.Context, userID int64) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM user_scenario_state WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("delete scenario state of user %d: %w", userID, err)
	}
	return nil
}
