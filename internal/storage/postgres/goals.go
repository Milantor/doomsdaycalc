package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lab042.ru/doomsdaycalc/internal/domain"
)

// GoalRepo: PostgreSQL implementation of domain.GoalRepository.
type GoalRepo struct {
	pool *pgxpool.Pool
}

// NewGoalRepo: goal repo backed by the given pool.
func NewGoalRepo(pool *pgxpool.Pool) *GoalRepo {
	return &GoalRepo{pool: pool}
}

// GoalRepo implements domain.GoalRepository.
var _ domain.GoalRepository = (*GoalRepo)(nil)

// goalCols: shared column list, so every query scans the same order.
const goalCols = `id, user_id, title, deadline, timezone, currency, savings_currency, target_min, target_ok, target_max, active, created_at`

// scanGoal: reads one goal row, folding the three target columns back into the map.
// Takes a pgx.Row, which both QueryRow and a Rows cursor satisfy, so Get and
// ListByUser share the scanning code.
func scanGoal(row pgx.Row) (domain.Goal, error) {
	var (
		g               domain.Goal
		tMin, tOK, tMax int64
	)
	if err := row.Scan(&g.ID, &g.UserID, &g.Title, &g.Deadline, &g.Timezone,
		&g.Currency, &g.SavingsCurrency, &tMin, &tOK, &tMax, &g.Active, &g.CreatedAt); err != nil {
		return domain.Goal{}, err
	}
	g.Targets = map[domain.Tier]domain.Money{
		domain.TierMinimal:    domain.Money(tMin),
		domain.TierAcceptable: domain.Money(tOK),
		domain.TierOptimal:    domain.Money(tMax),
	}
	return g, nil
}

// Create inserts a goal and returns it with the assigned id and created_at.
func (r *GoalRepo) Create(ctx context.Context, g domain.Goal) (domain.Goal, error) {
	const q = `
INSERT INTO goals (user_id, title, deadline, timezone, currency, savings_currency, target_min, target_ok, target_max, active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING ` + goalCols

	out, err := scanGoal(r.pool.QueryRow(ctx, q,
		g.UserID, g.Title, g.Deadline, g.Timezone,
		g.Currency, g.SavingsCurrency,
		g.Targets[domain.TierMinimal],
		g.Targets[domain.TierAcceptable],
		g.Targets[domain.TierOptimal],
		g.Active,
	))
	if err != nil {
		return domain.Goal{}, fmt.Errorf("create goal: %w", err)
	}
	return out, nil
}

// Get returns a single goal by id, or domain.ErrNotFound.
func (r *GoalRepo) Get(ctx context.Context, id int64) (domain.Goal, error) {
	q := `SELECT ` + goalCols + ` FROM goals WHERE id = $1`

	g, err := scanGoal(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Goal{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Goal{}, fmt.Errorf("get goal %d: %w", id, err)
	}
	return g, nil
}

// ListByUser returns all goals of the user, oldest first.
func (r *GoalRepo) ListByUser(ctx context.Context, userID int64) ([]domain.Goal, error) {
	q := `SELECT ` + goalCols + ` FROM goals WHERE user_id = $1 ORDER BY created_at`

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list goals of user %d: %w", userID, err)
	}
	defer rows.Close()

	var out []domain.Goal
	for rows.Next() {
		g, err := scanGoal(rows)
		if err != nil {
			return nil, fmt.Errorf("scan goal: %w", err)
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate goals: %w", err)
	}
	return out, nil
}

// SetActive enables or disables a goal.
func (r *GoalRepo) SetActive(ctx context.Context, id int64, active bool) error {
	tag, err := r.pool.Exec(ctx, `UPDATE goals SET active = $2 WHERE id = $1`, id, active)
	if err != nil {
		return fmt.Errorf("set goal %d active=%v: %w", id, active, err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
