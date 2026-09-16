package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lab042.ru/doomsdaycalc/internal/domain"
)

// UserRepo: PostgreSQL implementation of domain.UserRepository.
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo: user repo backed by the given pool.
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// UserRepo implements domain.UserRepository.
var _ domain.UserRepository = (*UserRepo)(nil)

// userCols: shared column list, so every query scans the same order.
const userCols = `id, username, first_name, language_code, ui_language, created_at, updated_at`

// Upsert inserts the user or refreshes mutable profile fields on conflict.
// ui_language (manual override) is absent from the SET list: operator sets it and it
// must survive every profile refresh.
func (r *UserRepo) Upsert(ctx context.Context, u domain.User) (domain.User, error) {
	const q = `
INSERT INTO users (id, username, first_name, language_code)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE
SET username      = EXCLUDED.username,
    first_name    = EXCLUDED.first_name,
    language_code = EXCLUDED.language_code,
    updated_at    = now()
RETURNING ` + userCols

	var out domain.User
	err := r.pool.QueryRow(ctx, q, u.ID, u.Username, u.FirstName, u.LanguageCode).
		Scan(&out.ID, &out.Username, &out.FirstName, &out.LanguageCode, &out.UILanguage, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		return domain.User{}, fmt.Errorf("upsert user %d: %w", u.ID, err)
	}
	return out, nil
}

// Get returns the user with the given Telegram id, or domain.ErrNotFound.
func (r *UserRepo) Get(ctx context.Context, id int64) (domain.User, error) {
	q := `SELECT ` + userCols + ` FROM users WHERE id = $1`

	var u domain.User
	err := r.pool.QueryRow(ctx, q, id).
		Scan(&u.ID, &u.Username, &u.FirstName, &u.LanguageCode, &u.UILanguage, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get user %d: %w", id, err)
	}
	return u, nil
}

// Delete hard-deletes the user. Related rows (goals, deposits, scenario state)
// cascade away.
func (r *UserRepo) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
