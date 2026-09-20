package scenario

import "lab042.ru/doomsdaycalc/internal/i18n"

// amountNodes: the single question shared by AddDeposit and Withdraw. Free text kept
// under VarAmount, with no Next, so the answer ends the dialogue and the service stores
// the deposit. Zero is a valid answer: it cancels the dialogue and stores nothing. A
// fresh map on each call, so the two scenarios never share mutable state.
func amountNodes(prompt func(i18n.Messages) string) map[string]Node {
	return map[string]Node{
		"ask_amount": {
			Prompt: prompt,
			Var:    VarAmount,
			Parse:  parseMoneyAmount,
		},
	}
}

// AddDeposit: collects one amount to put aside. The goal is picked before the dialogue
// starts and seeded into Vars under VarGoal, so the graph stays static and shared.
var AddDeposit = Scenario{
	Name:   "add_deposit",
	Entry:  "ask_amount",
	Result: ResultDeposit,
	Nodes:  amountNodes(func(m i18n.Messages) string { return m.AskDepositAmount }),
}

// Withdraw: same shape as AddDeposit, but money is taken back. The service negates the
// amount when ResultWithdraw is reported.
var Withdraw = Scenario{
	Name:   "withdraw",
	Entry:  "ask_amount",
	Result: ResultWithdraw,
	Nodes:  amountNodes(func(m i18n.Messages) string { return m.AskWithdrawAmount }),
}
