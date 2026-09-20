package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"lab042.ru/doomsdaycalc/internal/domain"
	"lab042.ru/doomsdaycalc/internal/scenario"
)

// ScenarioService: moves a user through a dialogue. Holds the state repository and the
// savings service; the graph data sits in internal/scenario. What happens on finish
// depends on Scenario.Result, so a scenario about jokes/polls/notifications only
// clears the state.
type ScenarioService struct {
	states  domain.ScenarioStateRepository
	savings *SavingsService
}

// NewScenarioService: scenario service backed by the given state repository and the
// savings service that stores finished goals.
func NewScenarioService(states domain.ScenarioStateRepository, savings *SavingsService) *ScenarioService {
	return &ScenarioService{states: states, savings: savings}
}

// Begin: starts a scenario for the user and returns its first node. The entry node is
// saved as the position, so a restart resumes right there. Vars start empty.
func (s *ScenarioService) Begin(ctx context.Context, userID int64, name string, now time.Time) (scenario.Node, error) {
	return s.BeginWithVars(ctx, userID, name, nil, now)
}

// BeginWithVars: same as Begin, but some answers are already filled in. Needed when a
// value is picked before the dialogue opens, e.g. the goal of a deposit.
func (s *ScenarioService) BeginWithVars(ctx context.Context, userID int64, name string, vars map[string]string, now time.Time) (scenario.Node, error) {
	sc, ok := scenario.ByName(name)
	if !ok {
		return scenario.Node{}, fmt.Errorf("%w: %q", ErrUnknownScenario, name)
	}
	if vars == nil {
		vars = map[string]string{}
	}
	if err := s.states.Save(ctx, domain.ScenarioState{
		UserID:       userID,
		ScenarioName: sc.Name,
		NodeID:       sc.Entry,
		EnteredAt:    now,
		Vars:         vars,
	}); err != nil {
		return scenario.Node{}, err
	}
	return sc.Nodes[sc.Entry], nil
}

// Current: the pending dialogue of the user. ok is false when no dialogue is running.
// Also returns the saved state, so Answer reuses it and reads the repository only once.
// A saved position that no longer maps to a node (scenario renamed, node removed) is
// dropped, so the user never sticks.
func (s *ScenarioService) Current(ctx context.Context, userID int64) (domain.ScenarioState, scenario.Node, bool, error) {
	st, err := s.states.Get(ctx, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.ScenarioState{}, scenario.Node{}, false, nil
	}
	if err != nil {
		return domain.ScenarioState{}, scenario.Node{}, false, err
	}

	node, ok := nodeOf(st)
	if !ok {
		if derr := s.Abort(ctx, userID); derr != nil {
			return domain.ScenarioState{}, scenario.Node{}, false, derr
		}
		return domain.ScenarioState{}, scenario.Node{}, false, nil
	}
	return st, node, true, nil
}

// Answer: feeds one answer into the running dialogue. st is the pending state that
// Current loaded, so the position is not read again. running is false when the dialogue
// just ended; the work asked for by Scenario.Result has already run. While the dialogue
// is still on, an error means the answer did not fit and the same node is asked again.
func (s *ScenarioService) Answer(ctx context.Context, st domain.ScenarioState, answer string, now time.Time) (next scenario.Node, running bool, err error) {
	sc, ok := scenario.ByName(st.ScenarioName)
	if !ok {
		return scenario.Node{}, false, s.Abort(ctx, st.UserID)
	}
	node, ok := sc.Nodes[st.NodeID]
	if !ok {
		return scenario.Node{}, false, s.Abort(ctx, st.UserID)
	}

	nextID, value, serr := scenario.Step(sc, st.NodeID, answer)
	if serr != nil {
		return scenario.Node{}, true, serr
	}

	// Deadline rule needs the current time, which the FSM has no access to. The check
	// runs at the deadline node, so a date of today or earlier is refused and the
	// question is asked again.
	if node.Var == scenario.VarDeadline && value != "" {
		if derr := checkDeadline(value, now); derr != nil {
			return scenario.Node{}, true, fmt.Errorf("%w: %v", scenario.ErrBadAnswer, derr)
		}
	}

	// A withdrawal cannot take more than the goal holds. The check runs at the amount
	// node, so an answer over the balance is refused and the question is asked again.
	if sc.Result == scenario.ResultWithdraw && node.Var == scenario.VarAmount && value != "" {
		if derr := s.checkOverdraw(ctx, st.UserID, st.Vars, value); derr != nil {
			return scenario.Node{}, true, derr
		}
	}

	// Targets ascend minimal <= acceptable <= optimal, so a lower amount is refused at
	// the target node and the question is asked again.
	if value != "" {
		if below := tierBelow(node.Var); below != "" {
			if oerr := checkTierOrder(st.Vars[below], value); oerr != nil {
				return scenario.Node{}, true, fmt.Errorf("%w: %v", ErrTierOrder, oerr)
			}
		}
	}

	// Keep the answer under the key the node declared.
	if value != "" && node.Var != "" {
		if st.Vars == nil {
			st.Vars = map[string]string{}
		}
		st.Vars[node.Var] = value
	}

	// No next node: the dialogue is over, run the finish work, drop the position.
	if nextID == "" {
		finishErr := s.finish(ctx, sc, st, now)
		if derr := s.states.Delete(ctx, st.UserID); derr != nil {
			return scenario.Node{}, false, derr
		}
		return scenario.Node{}, false, finishErr
	}

	st.NodeID = nextID
	if serr := s.states.Save(ctx, st); serr != nil {
		return scenario.Node{}, true, serr
	}
	return sc.Nodes[nextID], true, nil
}

// Abort: drops the running dialogue. No error when none was running.
func (s *ScenarioService) Abort(ctx context.Context, userID int64) error {
	return s.states.Delete(ctx, userID)
}

// nodeOf: the node a saved position points at.
func nodeOf(st domain.ScenarioState) (scenario.Node, bool) {
	sc, ok := scenario.ByName(st.ScenarioName)
	if !ok {
		return scenario.Node{}, false
	}
	n, ok := sc.Nodes[st.NodeID]
	return n, ok
}

// finish: the work Scenario.Result asks for when a dialogue ends. ResultNone does
// nothing; ResultGoal stores a goal from the collected vars; ResultDeposit and
// ResultWithdraw store a signed deposit for the picked goal. A zero amount cancels the
// dialogue, so nothing is stored and ErrCancelled comes back.
func (s *ScenarioService) finish(ctx context.Context, sc *scenario.Scenario, st domain.ScenarioState, now time.Time) error {
	switch sc.Result {
	case scenario.ResultGoal:
		goal, err := goalFromVars(st)
		if err != nil {
			return err
		}
		_, err = s.savings.CreateGoal(ctx, goal, now)
		return err
	case scenario.ResultDeposit, scenario.ResultWithdraw:
		d, err := depositFromVars(st, sc.Result == scenario.ResultWithdraw)
		if err != nil {
			return err
		}
		// Zero amount cancels the dialogue, so nothing is stored.
		if d.Amount == 0 {
			return ErrCancelled
		}
		_, err = s.savings.AddDeposit(ctx, st.UserID, d.GoalID, d.Amount, now)
		return err
	default:
		return nil
	}
}

// checkDeadline: deadline must be at least tomorrow. A date of today or earlier is
// already past by the time the goal is stored, so the question is asked again. now
// comes in as a parameter, so the rule stays testable.
func checkDeadline(day string, now time.Time) error {
	d, err := time.Parse("2006-01-02", day)
	if err != nil {
		return err
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if !d.After(today) {
		return fmt.Errorf("deadline %q is not after today", day)
	}
	return nil
}

// checkOverdraw: a withdrawal cannot take more than the goal holds. amount is whole
// units while Balance is minor units, so amount is scaled by 100 first. A negative
// balance counts as nothing saved. A missing goal id is left to finish, which reports it
// and ends the dialogue, so the amount question is not repeated forever.
func (s *ScenarioService) checkOverdraw(ctx context.Context, userID int64, vars map[string]string, amount string) error {
	goalID, err := strconv.ParseInt(strings.TrimSpace(vars[scenario.VarGoal]), 10, 64)
	if err != nil {
		return nil
	}
	saved, err := s.savings.Balance(ctx, userID, goalID)
	if err != nil {
		return err
	}
	if saved < 0 {
		saved = 0
	}
	requested, err := parseAmount(amount)
	if err != nil {
		return err
	}
	if requested > saved {
		return fmt.Errorf("%w: withdraw %d over saved %d", ErrOverdraw, requested, saved)
	}
	return nil
}

// tierBelow: the dialogue var of the tier just under varKey, empty when varKey is the
// first tier or not a target.
func tierBelow(varKey string) string {
	switch varKey {
	case scenario.VarTargetOK:
		return scenario.VarTargetMin
	case scenario.VarTargetMax:
		return scenario.VarTargetOK
	default:
		return ""
	}
}

// checkTierOrder: targets ascend minimal <= acceptable <= optimal, so a value below the
// tier before it is refused. Both are whole numbers normalized by the node parser; a
// missing or bad below value is left to the finish step.
func checkTierOrder(below, value string) error {
	if below == "" {
		return nil
	}
	prev, perr := strconv.ParseInt(below, 10, 64)
	cur, cerr := strconv.ParseInt(value, 10, 64)
	if perr != nil || cerr != nil {
		return nil
	}
	if cur < prev {
		return fmt.Errorf("target %s is below %s", value, below)
	}
	return nil
}

// goalFromVars: builds a domain.Goal out of dialogue answers. Amounts are typed in
// whole rubles and Money is minor units, so they are scaled by 100. Currency is RUB on
// both sides until the conversion step lands; timezone is UTC. Deadline is end of the
// given day UTC, so picking today still counts as ahead.
func goalFromVars(st domain.ScenarioState) (domain.Goal, error) {
	day, err := time.Parse("2006-01-02", strings.TrimSpace(st.Vars[scenario.VarDeadline]))
	if err != nil {
		return domain.Goal{}, fmt.Errorf("%w: deadline %q", domain.ErrInvalidGoal, st.Vars[scenario.VarDeadline])
	}

	targets := make(map[domain.Tier]domain.Money, len(domain.Tiers))
	for _, t := range domain.Tiers {
		m, aerr := parseAmount(st.Vars[targetVar(t)])
		if aerr != nil {
			return domain.Goal{}, aerr
		}
		targets[t] = m
	}

	return domain.Goal{
		UserID:          st.UserID,
		Title:           strings.TrimSpace(st.Vars[scenario.VarTitle]),
		Deadline:        day.Add(24*time.Hour - time.Nanosecond),
		Timezone:        "UTC",
		Currency:        domain.CurrencyRUB,
		SavingsCurrency: domain.CurrencyRUB,
		Targets:         targets,
		Active:          true,
	}, nil
}

// depositFromVars: builds a deposit out of dialogue answers. The goal id was seeded
// before the dialogue started; the amount is typed in whole units and scaled to minor
// units. withdraw flips the sign, so a withdrawal lands as a negative row.
func depositFromVars(st domain.ScenarioState, withdraw bool) (domain.Deposit, error) {
	goalID, err := strconv.ParseInt(strings.TrimSpace(st.Vars[scenario.VarGoal]), 10, 64)
	if err != nil {
		return domain.Deposit{}, fmt.Errorf("%w: goal %q", domain.ErrNotFound, st.Vars[scenario.VarGoal])
	}
	amount, err := parseAmount(st.Vars[scenario.VarAmount])
	if err != nil {
		return domain.Deposit{}, err
	}
	if withdraw {
		amount = -amount
	}
	return domain.Deposit{GoalID: goalID, Amount: amount}, nil
}

// targetVar: the dialogue var key that holds the amount for a tier.
func targetVar(t domain.Tier) string {
	switch t {
	case domain.TierMinimal:
		return scenario.VarTargetMin
	case domain.TierAcceptable:
		return scenario.VarTargetOK
	default:
		return scenario.VarTargetMax
	}
}

// parseAmount: whole rubles typed by the user, as minor units.
func parseAmount(raw string) (domain.Money, error) {
	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: amount %q", domain.ErrInvalidGoal, raw)
	}
	return domain.Money(n * 100), nil
}
