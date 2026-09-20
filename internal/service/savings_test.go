package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"lab042.ru/doomsdaycalc/internal/domain"
)

// fakeGoalRepo: in-memory domain.GoalRepository. Create records what it got and
// hands back an id, so tests can tell "stored" from "rejected before storage".
type fakeGoalRepo struct {
	created []domain.Goal
	nextID  int64
	err     error
}

func (f *fakeGoalRepo) Create(_ context.Context, g domain.Goal) (domain.Goal, error) {
	if f.err != nil {
		return domain.Goal{}, f.err
	}
	f.nextID++
	g.ID = f.nextID
	f.created = append(f.created, g)
	return g, nil
}

func (f *fakeGoalRepo) Get(_ context.Context, id int64) (domain.Goal, error) {
	for _, g := range f.created {
		if g.ID == id {
			return g, nil
		}
	}
	return domain.Goal{}, domain.ErrNotFound
}

func (f *fakeGoalRepo) ListByUser(_ context.Context, userID int64) ([]domain.Goal, error) {
	var out []domain.Goal
	for _, g := range f.created {
		if g.UserID == userID {
			out = append(out, g)
		}
	}
	return out, nil
}

func (f *fakeGoalRepo) SetActive(context.Context, int64, bool) error { return nil }

// fakeDepositRepo: in-memory domain.DepositRepository, so the savings and scenario
// tests can check what was written.
type fakeDepositRepo struct {
	added  []domain.Deposit
	nextID int64
}

func (f *fakeDepositRepo) Add(_ context.Context, d domain.Deposit) (domain.Deposit, error) {
	f.nextID++
	d.ID = f.nextID
	f.added = append(f.added, d)
	return d, nil
}

func (f *fakeDepositRepo) ListByGoal(_ context.Context, goalID int64) ([]domain.Deposit, error) {
	var out []domain.Deposit
	for _, d := range f.added {
		if d.GoalID == goalID {
			out = append(out, d)
		}
	}
	return out, nil
}

var testNow = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

func testGoal() domain.Goal {
	return domain.Goal{
		UserID:          1,
		Title:           "New laptop",
		Deadline:        time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		Timezone:        "UTC",
		Currency:        domain.CurrencyEUR,
		SavingsCurrency: domain.CurrencyRUB,
		Targets: map[domain.Tier]domain.Money{
			domain.TierMinimal:    10000,
			domain.TierAcceptable: 20000,
			domain.TierOptimal:    30000,
		},
		Active: true,
	}
}

// newSavings: service on a fake goal repo. CreateGoal reads goals only, so the
// deposits repo stays nil in these tests.
func newSavings(repo *fakeGoalRepo) *SavingsService {
	return NewSavingsService(repo, nil)
}

func TestCreateGoalStoresValidGoal(t *testing.T) {
	repo := &fakeGoalRepo{}
	svc := newSavings(repo)

	got, err := svc.CreateGoal(context.Background(), testGoal(), testNow)
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if got.ID == 0 {
		t.Error("CreateGoal returned a goal without id")
	}
	if len(repo.created) != 1 {
		t.Fatalf("repo got %d goals, want 1", len(repo.created))
	}
}

func TestCreateGoalRejectsInvalidGoal(t *testing.T) {
	repo := &fakeGoalRepo{}
	svc := newSavings(repo)

	g := testGoal()
	g.Title = ""

	_, err := svc.CreateGoal(context.Background(), g, testNow)
	if !errors.Is(err, domain.ErrInvalidGoal) {
		t.Fatalf("err = %v, want ErrInvalidGoal", err)
	}
	if len(repo.created) != 0 {
		t.Error("repo was called for an invalid goal")
	}
}

func TestCreateGoalPropagatesRepoError(t *testing.T) {
	repo := &fakeGoalRepo{err: errors.New("boom")}
	svc := newSavings(repo)

	if _, err := svc.CreateGoal(context.Background(), testGoal(), testNow); err == nil {
		t.Fatal("CreateGoal swallowed the repo error")
	}
}

// statusGoal: a goal with a controlled window and targets, so status numbers are
// predictable without reading the clock.
func statusGoal() domain.Goal {
	return domain.Goal{
		ID:              7,
		UserID:          1,
		Title:           "Trip",
		CreatedAt:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Deadline:        time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC),
		Timezone:        "UTC",
		Currency:        domain.CurrencyRUB,
		SavingsCurrency: domain.CurrencyRUB,
		Targets: map[domain.Tier]domain.Money{
			domain.TierMinimal:    100000,
			domain.TierAcceptable: 200000,
			domain.TierOptimal:    300000,
		},
		Active: true,
	}
}

func TestAddDepositStoresForOwnGoal(t *testing.T) {
	goals := &fakeGoalRepo{}
	goals.created = append(goals.created, statusGoal())
	deposits := &fakeDepositRepo{}
	svc := NewSavingsService(goals, deposits)

	d, err := svc.AddDeposit(context.Background(), 1, 7, 50000, testNow)
	if err != nil {
		t.Fatalf("AddDeposit: %v", err)
	}
	if d.GoalID != 7 || d.Amount != 50000 {
		t.Errorf("deposit = {GoalID:%d Amount:%d}, want {7 50000}", d.GoalID, d.Amount)
	}
	if len(deposits.added) != 1 {
		t.Fatalf("repo got %d deposits, want 1", len(deposits.added))
	}
}

func TestAddDepositRejectsForeignGoal(t *testing.T) {
	goals := &fakeGoalRepo{}
	goals.created = append(goals.created, statusGoal()) // belongs to user 1
	deposits := &fakeDepositRepo{}
	svc := NewSavingsService(goals, deposits)

	_, err := svc.AddDeposit(context.Background(), 2, 7, 50000, testNow)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if len(deposits.added) != 0 {
		t.Error("repo was called for a foreign goal")
	}
}

func TestGoalStatusComputes(t *testing.T) {
	goals := &fakeGoalRepo{}
	goals.created = append(goals.created, statusGoal())
	deposits := &fakeDepositRepo{}
	deposits.added = append(deposits.added, domain.Deposit{
		GoalID:     7,
		Amount:     70000,
		HappenedAt: time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC),
	})
	svc := NewSavingsService(goals, deposits)

	g, st, err := svc.GoalStatus(context.Background(), 1, 7, time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GoalStatus: %v", err)
	}
	if g.ID != 7 {
		t.Errorf("goal id = %d, want 7", g.ID)
	}
	if st.Saved != 70000 {
		t.Errorf("saved = %d, want 70000", st.Saved)
	}
	if !st.Tiers[domain.TierMinimal].OnTrack {
		t.Error("minimal tier is not on track")
	}
	if st.Tiers[domain.TierAcceptable].OnTrack {
		t.Error("acceptable tier should not be on track")
	}
}

func TestGoalStatusRejectsForeignGoal(t *testing.T) {
	goals := &fakeGoalRepo{}
	goals.created = append(goals.created, statusGoal()) // belongs to user 1
	svc := NewSavingsService(goals, &fakeDepositRepo{})

	if _, _, err := svc.GoalStatus(context.Background(), 2, 7, testNow); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

// --- Balance ---

func TestBalanceSumsDeposits(t *testing.T) {
	goals := &fakeGoalRepo{}
	goals.created = append(goals.created, statusGoal()) // user 1, goal 7
	deposits := &fakeDepositRepo{}
	deposits.added = append(deposits.added,
		domain.Deposit{GoalID: 7, Amount: 70000},
		domain.Deposit{GoalID: 7, Amount: -20000},
		domain.Deposit{GoalID: 7, Amount: 5000},
	)
	svc := NewSavingsService(goals, deposits)

	got, err := svc.Balance(context.Background(), 1, 7)
	if err != nil {
		t.Fatalf("Balance: %v", err)
	}
	if got != 55000 {
		t.Errorf("balance = %d, want 55000", got)
	}
}

func TestBalanceEmptyGoal(t *testing.T) {
	goals := &fakeGoalRepo{}
	goals.created = append(goals.created, statusGoal())
	svc := NewSavingsService(goals, &fakeDepositRepo{})

	got, err := svc.Balance(context.Background(), 1, 7)
	if err != nil {
		t.Fatalf("Balance: %v", err)
	}
	if got != 0 {
		t.Errorf("balance = %d, want 0", got)
	}
}

func TestBalanceIgnoresOtherGoals(t *testing.T) {
	goals := &fakeGoalRepo{}
	goals.created = append(goals.created, statusGoal()) // goal 7, user 1
	deposits := &fakeDepositRepo{}
	deposits.added = append(deposits.added,
		domain.Deposit{GoalID: 7, Amount: 10000},
		domain.Deposit{GoalID: 8, Amount: 999900},
	)
	svc := NewSavingsService(goals, deposits)

	got, err := svc.Balance(context.Background(), 1, 7)
	if err != nil {
		t.Fatalf("Balance: %v", err)
	}
	if got != 10000 {
		t.Errorf("balance = %d, want 10000", got)
	}
}

func TestBalanceKeepsNegativeSum(t *testing.T) {
	goals := &fakeGoalRepo{}
	goals.created = append(goals.created, statusGoal())
	deposits := &fakeDepositRepo{}
	deposits.added = append(deposits.added, domain.Deposit{GoalID: 7, Amount: -10000})
	svc := NewSavingsService(goals, deposits)

	got, err := svc.Balance(context.Background(), 1, 7)
	if err != nil {
		t.Fatalf("Balance: %v", err)
	}
	if got != -10000 {
		t.Errorf("balance = %d, want -10000", got)
	}
}

func TestBalanceRejectsForeignGoal(t *testing.T) {
	goals := &fakeGoalRepo{}
	goals.created = append(goals.created, statusGoal()) // belongs to user 1
	deposits := &fakeDepositRepo{}
	deposits.added = append(deposits.added, domain.Deposit{GoalID: 7, Amount: 999900})
	svc := NewSavingsService(goals, deposits)

	if _, err := svc.Balance(context.Background(), 2, 7); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestBalanceUnknownGoal(t *testing.T) {
	svc := NewSavingsService(&fakeGoalRepo{}, &fakeDepositRepo{})

	if _, err := svc.Balance(context.Background(), 1, 7); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
