package domain

import (
	"context"
	"time"
)

// User: a Telegram user known to the bot. ID is the Telegram user id and the
// primary key in the database.
type User struct {
	ID           int64
	Username     string
	FirstName    string
	LanguageCode string
	// UILanguage: explicit interface-language override. Empty means derive from
	// LanguageCode (Telegram hint). Upsert does not touch it, so a manual override
	// survives profile refreshes.
	UILanguage Lang
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// UserRepository: stores users. Implemented in storage/postgres.
type UserRepository interface {
	// Upsert inserts the user or refreshes mutable profile fields.
	Upsert(ctx context.Context, u User) (User, error)
	// Get returns the user with the given Telegram id, or ErrNotFound.
	Get(ctx context.Context, id int64) (User, error)
	// Delete hard-deletes the user and, via ON DELETE CASCADE, everything else
	// that belongs to them.
	Delete(ctx context.Context, id int64) error
}
