package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lab042.ru/doomsdaycalc/internal/domain"
)

// DepositRepo: PostgreSQL implementation of domain.DepositRepository.
type DepositRepo struct {
	pool *pgxpool.Pool
}

// NewDepositRepo: deposit repo backed by the given pool.
func NewDepositRepo(pool *pgxpool.Pool) *DepositRepo {
	return &DepositRepo{pool: pool}
}

// DepositRepo implements domain.DepositRepository.
var _ domain.DepositRepository = (*DepositRepo)(nil)

// Add appends a deposit. A zero HappenedAt is replaced with the current time.
func (r *DepositRepo) Add(ctx context.Context, d domain.Deposit) (domain.Deposit, error) {
	if d.HappenedAt.IsZero() {
		d.HappenedAt = time.Now().UTC()
	}

	const q = `
INSERT INTO deposits (goal_id, amount, happened_at, note)
VALUES ($1, $2, $3, $4)
RETURNING id, goal_id, amount, happened_at, note`

	var out domain.Deposit
	err := r.pool.QueryRow(ctx, q, d.GoalID, d.Amount, d.HappenedAt, d.Note).
		Scan(&out.ID, &out.GoalID, &out.Amount, &out.HappenedAt, &out.Note)
	if err != nil {
		return domain.Deposit{}, fmt.Errorf("add deposit: %w", err)
	}
	return out, nil
}

// ListByGoal returns deposits of a goal, oldest first.
func (r *DepositRepo) ListByGoal(ctx context.Context, goalID int64) ([]domain.Deposit, error) {
	const q = `
SELECT id, goal_id, amount, happened_at, note
FROM deposits
WHERE goal_id = $1
ORDER BY happened_at, id`

	rows, err := r.pool.Query(ctx, q, goalID)
	if err != nil {
		return nil, fmt.Errorf("list deposits of goal %d: %w", goalID, err)
	}
	defer rows.Close()

	var out []domain.Deposit
	for rows.Next() {
		var d domain.Deposit
		if err := rows.Scan(&d.ID, &d.GoalID, &d.Amount, &d.HappenedAt, &d.Note); err != nil {
			return nil, fmt.Errorf("scan deposit: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deposits: %w", err)
	}
	return out, nil
}
