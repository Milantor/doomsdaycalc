package service

import (
	"context"
	"time"

	"lab042.ru/doomsdaycalc/internal/domain"
)

// SavingsService: app actions on goals and deposits. Holds both repositories, so
// bot never touches them directly. CreateGoal validates here, at the boundary;
// ComputeStatus stays pure in domain. One instance per process, wired in main and
// passed to bot through Deps.
type SavingsService struct {
	goals    domain.GoalRepository
	deposits domain.DepositRepository
}

// NewSavingsService: savings service backed by the given repositories.
func NewSavingsService(goals domain.GoalRepository, deposits domain.DepositRepository) *SavingsService {
	return &SavingsService{goals: goals, deposits: deposits}
}

// CreateGoal: validates the goal and stores it, returning the record with its id.
// `now` goes straight to domain.ValidateGoal; the same value later feeds status
// calculation.
func (s *SavingsService) CreateGoal(ctx context.Context, g domain.Goal, now time.Time) (domain.Goal, error) {
	if err := domain.ValidateGoal(g, now); err != nil {
		return domain.Goal{}, err
	}
	return s.goals.Create(ctx, g)
}

// ListGoals: every goal of the user, oldest first. Empty means the user has none yet,
// which is what starts onboarding.
func (s *SavingsService) ListGoals(ctx context.Context, userID int64) ([]domain.Goal, error) {
	return s.goals.ListByUser(ctx, userID)
}

// AddDeposit: stores one deposit for a goal of the user. The goal must belong to the
// user, otherwise domain.ErrNotFound, so a forged goal id from a callback cannot
// write into someone elses goal. Amount is signed: negative means money taken back.
func (s *SavingsService) AddDeposit(ctx context.Context, userID, goalID int64, amount domain.Money, happened time.Time) (domain.Deposit, error) {
	g, err := s.goals.Get(ctx, goalID)
	if err != nil {
		return domain.Deposit{}, err
	}
	if g.UserID != userID {
		return domain.Deposit{}, domain.ErrNotFound
	}
	return s.deposits.Add(ctx, domain.Deposit{GoalID: goalID, Amount: amount, HappenedAt: happened})
}

// Balance: total saved on one goal of the user, the sum of its signed deposits. A goal
// of another user gives domain.ErrNotFound.
func (s *SavingsService) Balance(ctx context.Context, userID, goalID int64) (domain.Money, error) {
	g, err := s.goals.Get(ctx, goalID)
	if err != nil {
		return 0, err
	}
	if g.UserID != userID {
		return 0, domain.ErrNotFound
	}
	deposits, err := s.deposits.ListByGoal(ctx, goalID)
	if err != nil {
		return 0, err
	}
	var sum domain.Money
	for _, d := range deposits {
		sum += d.Amount
	}
	return sum, nil
}

// GoalStatus: progress of one goal of the user at `now`. The rate stays identity until
// the currency step lands, so targets and deposits share the currency. The goal must
// belong to the user, otherwise domain.ErrNotFound.
func (s *SavingsService) GoalStatus(ctx context.Context, userID, goalID int64, now time.Time) (domain.Goal, domain.Status, error) {
	g, err := s.goals.Get(ctx, goalID)
	if err != nil {
		return domain.Goal{}, domain.Status{}, err
	}
	if g.UserID != userID {
		return domain.Goal{}, domain.Status{}, domain.ErrNotFound
	}
	deposits, err := s.deposits.ListByGoal(ctx, goalID)
	if err != nil {
		return domain.Goal{}, domain.Status{}, err
	}
	return g, domain.ComputeStatus(g, deposits, domain.IdentityRate(), now), nil
}
