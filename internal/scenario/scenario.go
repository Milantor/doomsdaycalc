// Package scenario: dialogue FSM. Dialogues are data (nodes, prompts, buttons), the
// engine is one pure Step function. Text comes from i18n through Text selectors, so
// literals stay in the catalog. No Telegram/SQL: the bot renders nodes and persists
// the position through domain.ScenarioStateRepository.
package scenario

import "lab042.ru/doomsdaycalc/internal/i18n"

// FreeText: Next key for a free-text answer. A button answer keys off its Data, free
// text has none, so it parks under this one.
const FreeText = ""

// Text: reads one string out of a language catalog. Nodes hold Text, so the literals
// stay in i18n and a renamed field breaks the build.
type Text func(i18n.Messages) string

// Button: one reply option under a node. Data is the key into Node.Next and the
// value compared against the incoming answer. Value is stored under Node.Var when
// this button is pressed; empty Value stores nothing.
type Button struct {
	Label Text
	Data  string
	Value string
}

// ButtonData: the Data of the button whose label matches the answer. A reply-keyboard
// button sends its label, while Step matches on Data, so the label is resolved here,
// next to the buttons. An answer matching nothing comes back unchanged, so Step can
// reject it.
func (n Node) ButtonData(m i18n.Messages, answer string) string {
	for _, b := range n.Buttons {
		if b.Label(m) == answer {
			return b.Data
		}
	}
	return answer
}

// Node: one step of a dialogue.
// Prompt is the message the node sends. Var names the key a free-text answer is kept
// under, empty when the node takes buttons only. Parse normalizes a free-text answer
// (a date, an amount) and rejects what does not fit. Next maps an answer key
// (Button.Data or FreeText) to the following node id; a missing key ends the dialogue.
type Node struct {
	Prompt  Text
	Buttons []Button
	Var     string
	Parse   func(string) (string, error)
	Next    map[string]string
}

// Result: what a finished dialogue does with its vars. Most scenarios (jokes,
// polls, multi-step notifications) produce nothing and leave savings alone.
type Result int

const (
	ResultNone     Result = iota // clear the state, nothing else
	ResultGoal                   // vars describe a goal, the service builds and stores it
	ResultDeposit                // vars hold a goal id and an amount, money goes in
	ResultWithdraw               // same vars as ResultDeposit, the money goes out
)

// Scenario: a whole dialogue graph. Nodes is keyed by node id, Entry is the id of
// the first node. Result tells the service what to do once the dialogue is over.
type Scenario struct {
	Name   string
	Entry  string
	Result Result
	Nodes  map[string]Node
}

// Var keys used by goal scenarios. The service reads the same keys back when it
// builds a domain.Goal, so these are a contract between the graph and the service.
const (
	VarTitle     = "title"
	VarDeadline  = "deadline"
	VarTargetMin = "target_min"
	VarTargetOK  = "target_ok"
	VarTargetMax = "target_max"
	VarAmount    = "amount" // amount typed for a deposit/withdraw dialogue
	VarGoal      = "goal"   // goal picked before a deposit/withdraw dialogue
)
