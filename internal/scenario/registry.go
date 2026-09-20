package scenario

// All: every known scenario. A name is persisted in the database, so renaming one
// strands the dialogues already running on it.
var All = []*Scenario{&Onboarding, &AddGoal, &AddDeposit, &Withdraw}

// ByName: scenario with the given name, or false when nothing matches.
func ByName(name string) (*Scenario, bool) {
	for _, s := range All {
		if s.Name == name {
			return s, true
		}
	}
	return nil, false
}
