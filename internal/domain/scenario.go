package domain

// This file (scenario.go) will hold the types for the scenario dialogue engine
// (the FSM that powers onboarding and later event dialogues):
//
//	type Button struct{ Text, Data string }
//	type Node struct {
//	    ID      string
//	    Text    string            // message template
//	    Buttons []Button          // reply options
//	    Next    map[string]string // button data -> next node id
//	}
//	type Scenario struct {
//	    Name  string
//	    Entry string // id of the entry node
//	    Nodes map[string]Node
//	}
//
// Scenarios are described as data, not code, so new dialogues can be added
// without touching the router in internal/bot. A user's current scenario and node
// are persisted in the database, not kept in memory.
