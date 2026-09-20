package service

import (
	"context"
	"errors"
	"testing"

	"lab042.ru/doomsdaycalc/internal/domain"
	"lab042.ru/doomsdaycalc/internal/scenario"
)

// fakeStateRepo: in-memory domain.ScenarioStateRepository, one position per user.
type fakeStateRepo struct {
	saved map[int64]domain.ScenarioState
}

func newFakeStateRepo() *fakeStateRepo {
	return &fakeStateRepo{saved: map[int64]domain.ScenarioState{}}
}

func (f *fakeStateRepo) Get(_ context.Context, userID int64) (domain.ScenarioState, error) {
	st, ok := f.saved[userID]
	if !ok {
		return domain.ScenarioState{}, domain.ErrNotFound
	}
	return st, nil
}

func (f *fakeStateRepo) Save(_ context.Context, s domain.ScenarioState) error {
	f.saved[s.UserID] = s
	return nil
}

func (f *fakeStateRepo) Delete(_ context.Context, userID int64) error {
	delete(f.saved, userID)
	return nil
}

func newScenarioSvc() (*ScenarioService, *fakeStateRepo, *fakeGoalRepo) {
	states := newFakeStateRepo()
	goals := &fakeGoalRepo{}
	return NewScenarioService(states, NewSavingsService(goals, nil)), states, goals
}

const testUser = 1

// turn: runs one dialogue step the way the bot does: load the pending state, then
// answer it. Keeps the Answer tests free of the state plumbing.
func turn(t *testing.T, svc *ScenarioService, userID int64, text string) (scenario.Node, bool, error) {
	t.Helper()
	ctx := context.Background()
	st, _, ok, err := svc.Current(ctx, userID)
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if !ok {
		t.Fatalf("no dialogue running for user %d", userID)
	}
	return svc.Answer(ctx, st, text, testNow)
}

func TestScenarioServiceWalksOnboarding(t *testing.T) {
	svc, states, goals := newScenarioSvc()
	ctx := context.Background()

	node, err := svc.Begin(ctx, testUser, "onboarding", testNow)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if node.Var != "" || len(node.Buttons) != 1 {
		t.Fatalf("entry node = {Var:%q Buttons:%d}, want the intro node", node.Var, len(node.Buttons))
	}

	// Intro button, then the five questions. After the last answer the dialogue ends.
	answers := []string{"go", "New laptop", "2026-06-01", "10000", "20000", "30000"}
	wantVar := []string{
		scenario.VarTitle, scenario.VarDeadline,
		scenario.VarTargetMin, scenario.VarTargetOK, scenario.VarTargetMax,
	}

	for i, ans := range answers {
		next, running, aerr := turn(t, svc, testUser, ans)
		if aerr != nil {
			t.Fatalf("Answer(%q): %v", ans, aerr)
		}
		last := i == len(answers)-1
		if running != !last {
			t.Fatalf("Answer(%q) running = %v, want %v", ans, running, !last)
		}
		if running && next.Var != wantVar[i] {
			t.Fatalf("Answer(%q) landed on node with Var %q, want %q", ans, next.Var, wantVar[i])
		}
	}

	if _, ok := states.saved[testUser]; ok {
		t.Error("state still saved after the dialogue ended")
	}
	if len(goals.created) != 1 {
		t.Fatalf("stored %d goals, want 1", len(goals.created))
	}

	g := goals.created[0]
	if g.UserID != testUser || g.Title != "New laptop" {
		t.Errorf("goal = {UserID:%d Title:%q}, want {1 New laptop}", g.UserID, g.Title)
	}
	if g.Currency != domain.CurrencyRUB || g.SavingsCurrency != domain.CurrencyRUB {
		t.Errorf("goal currencies = %q/%q, want RUB/RUB", g.Currency, g.SavingsCurrency)
	}
	if got := g.Targets[domain.TierMinimal]; got != 10000*100 {
		t.Errorf("minimal target = %d, want %d", got, 10000*100)
	}
	if got := g.Targets[domain.TierOptimal]; got != 30000*100 {
		t.Errorf("optimal target = %d, want %d", got, 30000*100)
	}
	if !g.Deadline.After(testNow) {
		t.Errorf("deadline %v is not after %v", g.Deadline, testNow)
	}
}

// TestScenarioServiceIntroTakesAnyAnswer: the intro takes any text, so a user who types
// instead of tapping the button still moves on to the first question and stores nothing.
func TestScenarioServiceIntroTakesAnyAnswer(t *testing.T) {
	svc, states, _ := newScenarioSvc()
	ctx := context.Background()

	if _, err := svc.Begin(ctx, testUser, "onboarding", testNow); err != nil {
		t.Fatalf("Begin: %v", err)
	}

	next, running, err := turn(t, svc, testUser, "not a button")
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if !running {
		t.Fatal("dialogue ended on the intro")
	}
	if next.Var != scenario.VarTitle {
		t.Errorf("next node Var = %q, want %q", next.Var, scenario.VarTitle)
	}
	st := states.saved[testUser]
	if st.NodeID != "ask_title" {
		t.Errorf("position = %q, want ask_title", st.NodeID)
	}
	if len(st.Vars) != 0 {
		t.Errorf("intro stored vars %v, want none", st.Vars)
	}
}

func TestScenarioServiceCurrentAndAbort(t *testing.T) {
	svc, _, _ := newScenarioSvc()
	ctx := context.Background()

	if _, _, ok, err := svc.Current(ctx, testUser); err != nil || ok {
		t.Fatalf("Current before Begin = (%v, %v), want (false, nil)", ok, err)
	}

	if _, err := svc.Begin(ctx, testUser, "add_goal", testNow); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	_, node, ok, err := svc.Current(ctx, testUser)
	if err != nil || !ok {
		t.Fatalf("Current after Begin = (%v, %v), want (true, nil)", ok, err)
	}
	if node.Var != scenario.VarTitle {
		t.Errorf("Current Var = %q, want %q", node.Var, scenario.VarTitle)
	}

	if err := svc.Abort(ctx, testUser); err != nil {
		t.Fatalf("Abort: %v", err)
	}
	if _, _, ok, _ := svc.Current(ctx, testUser); ok {
		t.Error("dialogue still running after Abort")
	}
}

func TestScenarioServiceBeginUnknownScenario(t *testing.T) {
	svc, _, _ := newScenarioSvc()

	if _, err := svc.Begin(context.Background(), testUser, "nope", testNow); !errors.Is(err, ErrUnknownScenario) {
		t.Fatalf("err = %v, want ErrUnknownScenario", err)
	}
}

// TestScenarioServiceDropsStaleState: a scenario removed from the registry must not
// leave the user stuck.
func TestScenarioServiceDropsStaleState(t *testing.T) {
	svc, states, _ := newScenarioSvc()
	ctx := context.Background()

	states.saved[testUser] = domain.ScenarioState{UserID: testUser, ScenarioName: "gone", NodeID: "x"}

	if _, _, ok, err := svc.Current(ctx, testUser); err != nil || ok {
		t.Fatalf("Current = (%v, %v), want (false, nil)", ok, err)
	}
	if _, ok := states.saved[testUser]; ok {
		t.Error("stale state was not dropped")
	}
}

// TestScenarioServiceRejectsBadDate: the date question carries a parser, so a wrong
// answer must be refused on the spot, leaving the position on the same node.
func TestScenarioServiceRejectsBadDate(t *testing.T) {
	svc, states, goals := newScenarioSvc()
	ctx := context.Background()

	if _, err := svc.Begin(ctx, testUser, "add_goal", testNow); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if _, _, err := turn(t, svc, testUser, "New laptop"); err != nil {
		t.Fatalf("Answer(title): %v", err)
	}

	_, running, err := turn(t, svc, testUser, "not a date")
	if !errors.Is(err, scenario.ErrBadAnswer) {
		t.Fatalf("err = %v, want ErrBadAnswer", err)
	}
	if !running {
		t.Error("dialogue ended on a bad date")
	}
	if st := states.saved[testUser]; st.NodeID != "ask_deadline" {
		t.Errorf("position moved to %q, want ask_deadline", st.NodeID)
	}
	if len(goals.created) != 0 {
		t.Error("a goal was stored from a bad date")
	}
}

// TestScenarioServiceRejectsLowerTier: a target below the tier before it is refused at
// the target node, so the position stays and the earlier answers are kept. An equal
// amount is allowed.
func TestScenarioServiceRejectsLowerTier(t *testing.T) {
	svc, states, goals := newScenarioSvc()
	ctx := context.Background()

	if _, err := svc.Begin(ctx, testUser, "add_goal", testNow); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	for _, ans := range []string{"New laptop", "2026-06-01", "35", "42"} {
		if _, _, err := turn(t, svc, testUser, ans); err != nil {
			t.Fatalf("Answer(%q): %v", ans, err)
		}
	}

	// The optimal tier drops below the acceptable one.
	_, running, err := turn(t, svc, testUser, "35")
	if !errors.Is(err, ErrTierOrder) {
		t.Fatalf("err = %v, want ErrTierOrder", err)
	}
	if !running {
		t.Error("dialogue ended on a low tier")
	}
	if st := states.saved[testUser]; st.NodeID != "ask_target_max" {
		t.Errorf("position = %q, want ask_target_max", st.NodeID)
	}
	if len(goals.created) != 0 {
		t.Error("a goal was stored from a low tier")
	}

	// An amount equal to the tier before it passes and ends the dialogue.
	if _, running, err := turn(t, svc, testUser, "42"); err != nil || running {
		t.Fatalf("Answer = (%v, %v), want finished and no error", running, err)
	}
	if len(goals.created) != 1 {
		t.Fatalf("stored %d goals, want 1", len(goals.created))
	}
	if got := goals.created[0].Targets[domain.TierOptimal]; got != 42*100 {
		t.Errorf("optimal target = %d, want %d", got, 42*100)
	}
}

// TestScenarioServiceRejectsLowerMiddleTier: the acceptable tier is checked against the
// minimal one the same way.
func TestScenarioServiceRejectsLowerMiddleTier(t *testing.T) {
	svc, states, _ := newScenarioSvc()
	ctx := context.Background()

	if _, err := svc.Begin(ctx, testUser, "add_goal", testNow); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	for _, ans := range []string{"New laptop", "2026-06-01", "100"} {
		if _, _, err := turn(t, svc, testUser, ans); err != nil {
			t.Fatalf("Answer(%q): %v", ans, err)
		}
	}

	_, running, err := turn(t, svc, testUser, "50")
	if !errors.Is(err, ErrTierOrder) {
		t.Fatalf("err = %v, want ErrTierOrder", err)
	}
	if !running || states.saved[testUser].NodeID != "ask_target_ok" {
		t.Fatalf("position moved, want ask_target_ok still running")
	}
}

// TestGoalFromVars covers the assembly step directly. The dialogue parsers make these
// answers unlikely, but bad state must still fail loudly and store nothing.
func TestGoalFromVars(t *testing.T) {
	full := func() domain.ScenarioState {
		return domain.ScenarioState{
			UserID: testUser,
			Vars: map[string]string{
				scenario.VarTitle:     "New laptop",
				scenario.VarDeadline:  "2026-06-01",
				scenario.VarTargetMin: "10000",
				scenario.VarTargetOK:  "20000",
				scenario.VarTargetMax: "30000",
			},
		}
	}

	t.Run("good vars", func(t *testing.T) {
		g, err := goalFromVars(full())
		if err != nil {
			t.Fatalf("goalFromVars: %v", err)
		}
		if g.Title != "New laptop" || g.UserID != testUser {
			t.Errorf("goal = {UserID:%d Title:%q}", g.UserID, g.Title)
		}
		if got := g.Targets[domain.TierAcceptable]; got != 20000*100 {
			t.Errorf("acceptable target = %d, want %d", got, 20000*100)
		}
	})

	t.Run("bad date", func(t *testing.T) {
		st := full()
		st.Vars[scenario.VarDeadline] = "not a date"
		if _, err := goalFromVars(st); !errors.Is(err, domain.ErrInvalidGoal) {
			t.Fatalf("err = %v, want ErrInvalidGoal", err)
		}
	})

	t.Run("bad amount", func(t *testing.T) {
		st := full()
		st.Vars[scenario.VarTargetOK] = "20k"
		if _, err := goalFromVars(st); !errors.Is(err, domain.ErrInvalidGoal) {
			t.Fatalf("err = %v, want ErrInvalidGoal", err)
		}
	})
}

// newDepositSvc: scenario service on a fake state repo, one goal in the goal repo and a
// fake deposit repo, so deposit dialogues can be walked end to end.
func newDepositSvc() (*ScenarioService, *fakeStateRepo, *fakeDepositRepo) {
	states := newFakeStateRepo()
	goals := &fakeGoalRepo{}
	goals.created = append(goals.created, domain.Goal{ID: 7, UserID: testUser})
	deposits := &fakeDepositRepo{}
	return NewScenarioService(states, NewSavingsService(goals, deposits)), states, deposits
}

func TestScenarioServiceDepositFinish(t *testing.T) {
	svc, states, deposits := newDepositSvc()
	ctx := context.Background()

	if _, err := svc.BeginWithVars(ctx, testUser, "add_deposit", map[string]string{scenario.VarGoal: "7"}, testNow); err != nil {
		t.Fatalf("BeginWithVars: %v", err)
	}

	_, running, err := turn(t, svc, testUser, "500")
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if running {
		t.Error("dialogue still running after the amount")
	}
	if len(deposits.added) != 1 {
		t.Fatalf("stored %d deposits, want 1", len(deposits.added))
	}
	if d := deposits.added[0]; d.GoalID != 7 || d.Amount != 50000 {
		t.Errorf("deposit = {GoalID:%d Amount:%d}, want {7 50000}", d.GoalID, d.Amount)
	}
	if _, ok := states.saved[testUser]; ok {
		t.Error("state still saved after the dialogue ended")
	}
}

func TestScenarioServiceWithdrawFinish(t *testing.T) {
	svc, _, deposits := newDepositSvc()
	ctx := context.Background()
	deposits.added = append(deposits.added, domain.Deposit{GoalID: 7, Amount: 50000})

	if _, err := svc.BeginWithVars(ctx, testUser, "withdraw", map[string]string{scenario.VarGoal: "7"}, testNow); err != nil {
		t.Fatalf("BeginWithVars: %v", err)
	}

	if _, _, err := turn(t, svc, testUser, "500"); err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if len(deposits.added) != 2 {
		t.Fatalf("stored %d deposits, want 2", len(deposits.added))
	}
	if d := deposits.added[1]; d.Amount != -50000 {
		t.Errorf("withdraw amount = %d, want -50000", d.Amount)
	}
}

// TestScenarioServiceDepositWithoutGoal: a valid amount but no goal picked must fail
// loudly and store nothing.
func TestScenarioServiceDepositWithoutGoal(t *testing.T) {
	svc, states, deposits := newDepositSvc()

	states.saved[testUser] = domain.ScenarioState{
		UserID:       testUser,
		ScenarioName: "add_deposit",
		NodeID:       "ask_amount",
		Vars:         map[string]string{},
	}

	_, running, err := turn(t, svc, testUser, "500")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if running {
		t.Error("dialogue still running after a finish error")
	}
	if len(deposits.added) != 0 {
		t.Error("a deposit was stored without a goal")
	}
}

// TestScenarioServiceDeadlineRule: the deadline must be at least tomorrow. A date of
// today is refused at the deadline node and the position stays there; tomorrow passes.
func TestScenarioServiceDeadlineRule(t *testing.T) {
	svc, states, goals := newScenarioSvc()
	ctx := context.Background()

	if _, err := svc.Begin(ctx, testUser, "add_goal", testNow); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if _, _, err := turn(t, svc, testUser, "New laptop"); err != nil {
		t.Fatalf("Answer(title): %v", err)
	}

	// testNow is 2024-01-01, so this date is today and must be refused.
	_, running, err := turn(t, svc, testUser, "2024-01-01")
	if !errors.Is(err, scenario.ErrBadAnswer) {
		t.Fatalf("today: err = %v, want ErrBadAnswer", err)
	}
	if !running || states.saved[testUser].NodeID != "ask_deadline" {
		t.Fatalf("today: position moved, want ask_deadline still running")
	}
	if len(goals.created) != 0 {
		t.Error("a goal was stored from a past date")
	}

	// Tomorrow passes and the dialogue moves to the first target.
	next, running, err := turn(t, svc, testUser, "2024-01-02")
	if err != nil || !running {
		t.Fatalf("tomorrow: Answer = (%v, %v), want running and no error", running, err)
	}
	if next.Var != scenario.VarTargetMin {
		t.Errorf("tomorrow: next node Var = %q, want %q", next.Var, scenario.VarTargetMin)
	}
}

// TestScenarioServiceDepositCancel: a zero amount cancels the money dialogue, so nothing
// is stored and the position is dropped.
func TestScenarioServiceDepositCancel(t *testing.T) {
	svc, states, deposits := newDepositSvc()
	ctx := context.Background()

	if _, err := svc.BeginWithVars(ctx, testUser, "add_deposit", map[string]string{scenario.VarGoal: "7"}, testNow); err != nil {
		t.Fatalf("BeginWithVars: %v", err)
	}

	_, running, err := turn(t, svc, testUser, "0")
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("err = %v, want ErrCancelled", err)
	}
	if running {
		t.Error("dialogue still running after a cancel")
	}
	if len(deposits.added) != 0 {
		t.Errorf("stored %d deposits, want 0", len(deposits.added))
	}
	if _, ok := states.saved[testUser]; ok {
		t.Error("state still saved after a cancel")
	}
}

// --- overdraw ---

// TestScenarioServiceWithdrawOverdraw: a withdrawal over the saved total is refused at
// the amount node, so the position stays and nothing is stored. The exact saved total
// passes.
func TestScenarioServiceWithdrawOverdraw(t *testing.T) {
	svc, states, deposits := newDepositSvc()
	ctx := context.Background()
	deposits.added = append(deposits.added, domain.Deposit{GoalID: 7, Amount: 50000})

	if _, err := svc.BeginWithVars(ctx, testUser, "withdraw", map[string]string{scenario.VarGoal: "7"}, testNow); err != nil {
		t.Fatalf("BeginWithVars: %v", err)
	}

	// 1000 whole units is 100000 minor, over the saved 50000.
	_, running, err := turn(t, svc, testUser, "1000")
	if !errors.Is(err, ErrOverdraw) {
		t.Fatalf("err = %v, want ErrOverdraw", err)
	}
	if !running {
		t.Error("dialogue ended on an overdraw")
	}
	if st := states.saved[testUser]; st.NodeID != "ask_amount" {
		t.Errorf("position moved to %q, want ask_amount", st.NodeID)
	}
	if len(deposits.added) != 1 {
		t.Fatalf("stored %d deposits, want only the seeded one", len(deposits.added))
	}

	// The exact saved total is allowed.
	_, running, err = turn(t, svc, testUser, "500")
	if err != nil || running {
		t.Fatalf("exact: Answer = (%v, %v), want finished and no error", running, err)
	}
	if got := deposits.added[1].Amount; got != -50000 {
		t.Errorf("withdraw amount = %d, want -50000", got)
	}
	if _, ok := states.saved[testUser]; ok {
		t.Error("state still saved after the dialogue ended")
	}
}

// TestScenarioServiceWithdrawEmptyGoal: with nothing saved, any positive withdrawal is
// refused.
func TestScenarioServiceWithdrawEmptyGoal(t *testing.T) {
	svc, states, deposits := newDepositSvc()
	ctx := context.Background()

	if _, err := svc.BeginWithVars(ctx, testUser, "withdraw", map[string]string{scenario.VarGoal: "7"}, testNow); err != nil {
		t.Fatalf("BeginWithVars: %v", err)
	}

	_, running, err := turn(t, svc, testUser, "1")
	if !errors.Is(err, ErrOverdraw) {
		t.Fatalf("err = %v, want ErrOverdraw", err)
	}
	if !running || states.saved[testUser].NodeID != "ask_amount" {
		t.Fatalf("position moved, want ask_amount still running")
	}
	if len(deposits.added) != 0 {
		t.Errorf("stored %d deposits, want 0", len(deposits.added))
	}
}

// TestScenarioServiceWithdrawNegativeBalance: legacy data can leave the sum below zero.
// A negative balance counts as nothing saved, so a positive withdrawal is refused while a
// zero still cancels.
func TestScenarioServiceWithdrawNegativeBalance(t *testing.T) {
	svc, _, deposits := newDepositSvc()
	ctx := context.Background()
	deposits.added = append(deposits.added, domain.Deposit{GoalID: 7, Amount: -10000})

	if _, err := svc.BeginWithVars(ctx, testUser, "withdraw", map[string]string{scenario.VarGoal: "7"}, testNow); err != nil {
		t.Fatalf("BeginWithVars: %v", err)
	}

	if _, _, err := turn(t, svc, testUser, "1"); !errors.Is(err, ErrOverdraw) {
		t.Fatalf("positive: err = %v, want ErrOverdraw", err)
	}

	// Zero cancels even with the balance below zero.
	_, running, err := turn(t, svc, testUser, "0")
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("zero: err = %v, want ErrCancelled", err)
	}
	if running {
		t.Error("dialogue still running after a cancel")
	}
	if len(deposits.added) != 1 {
		t.Errorf("stored %d deposits, want only the seeded one", len(deposits.added))
	}
}

// TestScenarioServiceWithdrawCountsPickedGoalOnly: deposits on another goal do not raise
// the balance of the picked one.
func TestScenarioServiceWithdrawCountsPickedGoalOnly(t *testing.T) {
	svc, _, deposits := newDepositSvc()
	ctx := context.Background()
	deposits.added = append(deposits.added,
		domain.Deposit{GoalID: 7, Amount: 10000},
		domain.Deposit{GoalID: 8, Amount: 999900},
	)

	if _, err := svc.BeginWithVars(ctx, testUser, "withdraw", map[string]string{scenario.VarGoal: "7"}, testNow); err != nil {
		t.Fatalf("BeginWithVars: %v", err)
	}

	// 200 whole units is 20000 minor, over the 10000 saved on goal 7.
	if _, _, err := turn(t, svc, testUser, "200"); !errors.Is(err, ErrOverdraw) {
		t.Fatalf("err = %v, want ErrOverdraw", err)
	}
}

// TestScenarioServiceWithdrawKeepsVarGoal: an overdraw leaves the seeded goal answer in
// place and does not store the amount, so the same question can be answered again.
func TestScenarioServiceWithdrawKeepsVarGoal(t *testing.T) {
	svc, states, deposits := newDepositSvc()
	ctx := context.Background()
	deposits.added = append(deposits.added, domain.Deposit{GoalID: 7, Amount: 10000})

	if _, err := svc.BeginWithVars(ctx, testUser, "withdraw", map[string]string{scenario.VarGoal: "7"}, testNow); err != nil {
		t.Fatalf("BeginWithVars: %v", err)
	}

	if _, _, err := turn(t, svc, testUser, "500"); !errors.Is(err, ErrOverdraw) {
		t.Fatalf("err = %v, want ErrOverdraw", err)
	}

	st := states.saved[testUser]
	if st.Vars[scenario.VarGoal] != "7" {
		t.Errorf("VarGoal = %q, want 7", st.Vars[scenario.VarGoal])
	}
	if _, ok := st.Vars[scenario.VarAmount]; ok {
		t.Error("amount was stored on an overdraw")
	}
}

// TestScenarioServiceDepositHasNoCeiling: the balance rule guards withdrawals only, so a
// deposit dialogue stores any positive amount.
func TestScenarioServiceDepositHasNoCeiling(t *testing.T) {
	svc, _, deposits := newDepositSvc()
	ctx := context.Background()

	if _, err := svc.BeginWithVars(ctx, testUser, "add_deposit", map[string]string{scenario.VarGoal: "7"}, testNow); err != nil {
		t.Fatalf("BeginWithVars: %v", err)
	}

	if _, running, err := turn(t, svc, testUser, "999999"); err != nil || running {
		t.Fatalf("Answer = (%v, %v), want finished and no error", running, err)
	}
	if len(deposits.added) != 1 || deposits.added[0].Amount != 99999900 {
		t.Fatalf("deposits = %+v, want one 99999900", deposits.added)
	}
}

// TestScenarioServiceWithdrawWithoutGoal: a withdrawal with no goal picked is left to
// finish, which fails loudly and drops the position, so the question is not repeated.
func TestScenarioServiceWithdrawWithoutGoal(t *testing.T) {
	svc, states, deposits := newDepositSvc()

	states.saved[testUser] = domain.ScenarioState{
		UserID:       testUser,
		ScenarioName: "withdraw",
		NodeID:       "ask_amount",
		Vars:         map[string]string{},
	}

	_, running, err := turn(t, svc, testUser, "500")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if running {
		t.Error("dialogue still running without a goal")
	}
	if len(deposits.added) != 0 {
		t.Error("a deposit was stored without a goal")
	}
	if _, ok := states.saved[testUser]; ok {
		t.Error("state still saved after a finish error")
	}
}
