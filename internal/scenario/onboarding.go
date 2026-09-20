package scenario

import "lab042.ru/doomsdaycalc/internal/i18n"

// goalNodes: the question set shared by AddGoal and Onboarding. A fresh map on each
// call, so the two scenarios never share mutable state.
func goalNodes() map[string]Node {
	return map[string]Node{
		"ask_title": {
			Prompt: func(m i18n.Messages) string { return m.AskTitle },
			Var:    VarTitle,
			Parse:  parseTitle,
			Next:   map[string]string{FreeText: "ask_deadline"},
		},
		"ask_deadline": {
			Prompt: func(m i18n.Messages) string { return m.AskDeadline },
			Var:    VarDeadline,
			Parse:  parseDate,
			Next:   map[string]string{FreeText: "ask_target_min"},
		},
		"ask_target_min": {
			Prompt: func(m i18n.Messages) string { return m.AskTargetMin },
			Var:    VarTargetMin,
			Parse:  parseAmount,
			Next:   map[string]string{FreeText: "ask_target_ok"},
		},
		"ask_target_ok": {
			Prompt: func(m i18n.Messages) string { return m.AskTargetOK },
			Var:    VarTargetOK,
			Parse:  parseAmount,
			Next:   map[string]string{FreeText: "ask_target_max"},
		},
		"ask_target_max": {
			Prompt: func(m i18n.Messages) string { return m.AskTargetMax },
			Var:    VarTargetMax,
			Parse:  parseAmount,
			// No Next: this answer ends the dialogue.
		},
	}
}

// AddGoal: repeatable dialogue that collects one goal. Starts at the first question,
// so a user who already has goals can add another.
var AddGoal = Scenario{
	Name:   "add_goal",
	Entry:  "ask_title",
	Result: ResultGoal,
	Nodes:  goalNodes(),
}

// Onboarding: first-run dialogue, one definition for every user. The same questions
// as AddGoal with an intro node in front.
var Onboarding = Scenario{
	Name:   "onboarding",
	Entry:  "intro",
	Result: ResultGoal,
	Nodes:  onboardingNodes(),
}

// onboardingNodes: the shared questions plus an intro node any answer advances into
// them.
func onboardingNodes() map[string]Node {
	nodes := goalNodes()
	nodes["intro"] = Node{
		Prompt:  func(m i18n.Messages) string { return m.OnboardingIntro },
		Buttons: []Button{{Label: func(m i18n.Messages) string { return m.BtnStart }, Data: "go"}},
		Next:    map[string]string{"go": "ask_title", FreeText: "ask_title"},
	}
	return nodes
}
