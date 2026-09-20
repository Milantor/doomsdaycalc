// Package service: application layer. Holds the actions the bot triggers.
// A service uses repository interfaces from domain and pure domain functions.
// No Telegram/SQL/network. Dependencies point one way:
// bot -> service -> domain <- storage.
package service

import (
	"context"

	"lab042.ru/doomsdaycalc/internal/domain"
)

// UserService: app actions on a user record. Holds the user repository, so bot
// never touches it directly. One instance per process, wired in main and passed to
// bot through Deps.
type UserService struct {
	users domain.UserRepository
}

// NewUserService: user service backed by the given repository.
func NewUserService(users domain.UserRepository) *UserService {
	return &UserService{users: users}
}

// Touch records the current Telegram profile and returns the stored record.
// Returned record holds the ui_language override; Upsert leaves it alone, so an
// operator-set language survives every update.
func (s *UserService) Touch(ctx context.Context, u domain.User) (domain.User, error) {
	return s.users.Upsert(ctx, u)
}

// Get returns the stored user with the given Telegram id, or domain.ErrNotFound.
// Needed when a message is pushed into someone elses chat and the language has to be
// resolved from their stored profile.
func (s *UserService) Get(ctx context.Context, id int64) (domain.User, error) {
	return s.users.Get(ctx, id)
}

// Delete removes the user. Related rows (goals, deposits, scenario state) cascade
// away.
func (s *UserService) Delete(ctx context.Context, id int64) error {
	return s.users.Delete(ctx, id)
}

// ListIDs returns the Telegram ids of every known user, for sending to everyone.
func (s *UserService) ListIDs(ctx context.Context) ([]int64, error) {
	return s.users.ListIDs(ctx)
}
