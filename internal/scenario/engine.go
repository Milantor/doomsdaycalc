package scenario

import "fmt"

// Step: advances the dialogue by one answer. Returns the next node id and the value
// to keep under the current node Var. Empty next id means the dialogue is over. Pure:
// no clock, no I/O.
func Step(s *Scenario, nodeID, answer string) (nextID, value string, err error) {
	node, ok := s.Nodes[nodeID]
	if !ok {
		return "", "", fmt.Errorf("%w: %q", ErrUnknownNode, nodeID)
	}

	// Button answer: Data picks the next node, Value is kept.
	for _, b := range node.Buttons {
		if b.Data == answer {
			return node.Next[b.Data], b.Value, nil
		}
	}

	// Free-text answer: kept under Var, normalized by Parse. A node without Var takes any
	// text when it has a FreeText transition, storing nothing.
	if node.Var == "" {
		next, ok := node.Next[FreeText]
		if !ok {
			return "", "", fmt.Errorf("%w: %q", ErrUnexpectedAnswer, answer)
		}
		return next, "", nil
	}

	value = answer
	if node.Parse != nil {
		parsed, perr := node.Parse(answer)
		if perr != nil {
			return "", "", fmt.Errorf("%w: %s", ErrBadAnswer, perr)
		}
		value = parsed
	}
	return node.Next[FreeText], value, nil
}
