package scenario

import (
	"errors"
	"testing"

	"lab042.ru/doomsdaycalc/internal/i18n"
)

// goalAnswers: the answers both goal scenarios expect, in order.
var goalAnswers = []struct {
	answer    string
	wantNext  string
	wantValue string
}{
	{"New laptop", "ask_deadline", "New laptop"},
	{"2026-06-01", "ask_target_min", "2026-06-01"},
	{"10000", "ask_target_ok", "10000"},
	{"20000", "ask_target_max", "20000"},
	{"30000", "", "30000"}, // last question, no Next: dialogue over
}

// walk: feeds goalAnswers one by one and checks every hop from `from`.
func walk(t *testing.T, sc *Scenario, from string) {
	t.Helper()
	nodeID := from
	for _, s := range goalAnswers {
		next, value, err := Step(sc, nodeID, s.answer)
		if err != nil {
			t.Fatalf("Step(%q, %q): %v", nodeID, s.answer, err)
		}
		if next != s.wantNext {
			t.Fatalf("Step(%q, %q) next = %q, want %q", nodeID, s.answer, next, s.wantNext)
		}
		if value != s.wantValue {
			t.Fatalf("Step(%q, %q) value = %q, want %q", nodeID, s.answer, value, s.wantValue)
		}
		nodeID = next
	}
}

func TestStepWalksAddGoal(t *testing.T) {
	walk(t, &AddGoal, AddGoal.Entry)
}

func TestStepWalksOnboardingQuestions(t *testing.T) {
	// Intro is separate; from ask_title on, Onboarding asks the same questions.
	walk(t, &Onboarding, "ask_title")
}

func TestOnboardingIntroAdvancesOnButton(t *testing.T) {
	next, value, err := Step(&Onboarding, Onboarding.Entry, "go")
	if err != nil {
		t.Fatalf("Step: %v", err)
	}
	if next != "ask_title" {
		t.Errorf("next = %q, want %q", next, "ask_title")
	}
	if value != "" {
		t.Errorf("value = %q, want empty (intro button stores nothing)", value)
	}
}

func TestStepButtonPicksNextAndKeepsValue(t *testing.T) {
	sc := Scenario{
		Name:  "test",
		Entry: "pick",
		Nodes: map[string]Node{
			"pick": {
				Var: "tier",
				Buttons: []Button{
					{Data: "min", Value: "min"},
					{Data: "max", Value: "max"},
				},
				Next: map[string]string{"min": "done", "max": "done"},
			},
			"done": {},
		},
	}

	next, value, err := Step(&sc, "pick", "max")
	if err != nil {
		t.Fatalf("Step: %v", err)
	}
	if next != "done" || value != "max" {
		t.Fatalf("Step = (%q, %q), want (\"done\", \"max\")", next, value)
	}
}

func TestStepErrors(t *testing.T) {
	sc := Scenario{
		Name:  "test",
		Entry: "pick",
		Nodes: map[string]Node{
			"pick": {Buttons: []Button{{Data: "go"}}, Next: map[string]string{"go": "done"}},
			"done": {},
		},
	}

	if _, _, err := Step(&sc, "nope", "go"); !errors.Is(err, ErrUnknownNode) {
		t.Errorf("unknown node: err = %v, want ErrUnknownNode", err)
	}
	if _, _, err := Step(&sc, "pick", "typo"); !errors.Is(err, ErrUnexpectedAnswer) {
		t.Errorf("unexpected answer: err = %v, want ErrUnexpectedAnswer", err)
	}
}

func TestStepParseNormalizesAndRejects(t *testing.T) {
	sc := Scenario{
		Name:  "test",
		Entry: "ask",
		Nodes: map[string]Node{
			"ask": {
				Var:   "amount",
				Parse: parseAmount,
				Next:  map[string]string{FreeText: "done"},
			},
			"done": {},
		},
	}

	if _, _, err := Step(&sc, "ask", "20k"); !errors.Is(err, ErrBadAnswer) {
		t.Errorf("err = %v, want ErrBadAnswer", err)
	}

	next, value, err := Step(&sc, "ask", "  20 ")
	if err != nil {
		t.Fatalf("Step: %v", err)
	}
	if next != "done" || value != "20" {
		t.Errorf("Step = (%q, %q), want (\"done\", \"20\")", next, value)
	}
}

// TestParseMoneyAmount: zero passes for a money dialogue, a negative number does not.
func TestParseMoneyAmount(t *testing.T) {
	cases := []struct {
		in   string
		want string
		bad  bool
	}{
		{"500", "500", false},
		{" 0 ", "0", false},
		{"0", "0", false},
		{"-1", "", true},
		{"abc", "", true},
	}
	for _, c := range cases {
		got, err := parseMoneyAmount(c.in)
		if c.bad {
			if err == nil {
				t.Errorf("parseMoneyAmount(%q) = %q, want error", c.in, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("parseMoneyAmount(%q) = (%q, %v), want (%q, nil)", c.in, got, err, c.want)
		}
	}
}

// TestStepMoneyNodeAcceptsZero: the money node ends the dialogue on zero, so the service
// sees the value and cancels.
func TestStepMoneyNodeAcceptsZero(t *testing.T) {
	next, value, err := Step(&AddDeposit, AddDeposit.Entry, "0")
	if err != nil {
		t.Fatalf("Step: %v", err)
	}
	if next != "" || value != "0" {
		t.Fatalf("Step = (%q, %q), want (\"\", \"0\")", next, value)
	}
}

// TestAllScenariosAreSound guards the data of every registered scenario: the entry
// exists, each Next target exists, each node has a prompt, each button has a label
// and a key to match on.
func TestAllScenariosAreSound(t *testing.T) {
	for _, sc := range All {
		if _, ok := sc.Nodes[sc.Entry]; !ok {
			t.Errorf("scenario %q: entry %q is missing", sc.Name, sc.Entry)
		}
		for id, n := range sc.Nodes {
			if n.Prompt == nil {
				t.Errorf("scenario %q: node %q has no prompt", sc.Name, id)
			}
			for key, target := range n.Next {
				if _, ok := sc.Nodes[target]; !ok {
					t.Errorf("scenario %q: node %q Next[%q] points at missing node %q", sc.Name, id, key, target)
				}
			}
			for _, b := range n.Buttons {
				if b.Label == nil {
					t.Errorf("scenario %q: node %q button %q has no label", sc.Name, id, b.Data)
				}
				if b.Data == "" {
					t.Errorf("scenario %q: node %q has a button with empty Data", sc.Name, id)
				}
			}
		}
	}
}

// TestGoalScenariosDoNotShareNodes: goalNodes returns a fresh map, so the intro added
// to Onboarding must not leak into AddGoal.
func TestGoalScenariosDoNotShareNodes(t *testing.T) {
	if _, ok := AddGoal.Nodes["intro"]; ok {
		t.Error("AddGoal has an intro node")
	}
	if _, ok := Onboarding.Nodes["intro"]; !ok {
		t.Error("Onboarding has no intro node")
	}
}

func TestNodeButtonData(t *testing.T) {
	node := Onboarding.Nodes[Onboarding.Entry]
	if len(node.Buttons) != 1 {
		t.Fatalf("intro has %d buttons, want 1", len(node.Buttons))
	}
	m := i18n.Get(i18n.Default)
	btn := node.Buttons[0]

	if got := node.ButtonData(m, btn.Label(m)); got != btn.Data {
		t.Errorf("ButtonData(label) = %q, want %q", got, btn.Data)
	}
	if got := node.ButtonData(m, "free text"); got != "free text" {
		t.Errorf("ButtonData(unmatched) = %q, want the answer unchanged", got)
	}
}
