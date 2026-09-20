package scenario

import "testing"

func TestByName(t *testing.T) {
	if _, ok := ByName("nope"); ok {
		t.Error("ByName(nope) found something")
	}
	for _, sc := range All {
		got, ok := ByName(sc.Name)
		if !ok || got != sc {
			t.Errorf("ByName(%q) did not return the registered scenario", sc.Name)
		}
	}
}

func TestScenarioNamesAreUnique(t *testing.T) {
	seen := make(map[string]bool, len(All))
	for _, sc := range All {
		if sc.Name == "" {
			t.Error("scenario with empty name in All")
			continue
		}
		if seen[sc.Name] {
			t.Errorf("duplicate scenario name %q", sc.Name)
		}
		seen[sc.Name] = true
	}
}

// TestGoalScenariosHaveResultGoal: both goal dialogues must report a goal, otherwise
// the service would throw their vars away on finish.
func TestGoalScenariosHaveResultGoal(t *testing.T) {
	for _, sc := range []*Scenario{&Onboarding, &AddGoal} {
		if sc.Result != ResultGoal {
			t.Errorf("scenario %q: Result = %v, want ResultGoal", sc.Name, sc.Result)
		}
	}
}

// TestDepositScenariosHaveTheirResult: the money dialogues must report their
// direction, otherwise the service cannot tell a deposit from a withdrawal.
func TestDepositScenariosHaveTheirResult(t *testing.T) {
	if AddDeposit.Result != ResultDeposit {
		t.Errorf("AddDeposit.Result = %v, want ResultDeposit", AddDeposit.Result)
	}
	if Withdraw.Result != ResultWithdraw {
		t.Errorf("Withdraw.Result = %v, want ResultWithdraw", Withdraw.Result)
	}
}
