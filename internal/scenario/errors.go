package scenario

import "errors"

// ErrUnknownNode: the scenario has no node with that id.
var ErrUnknownNode = errors.New("unknown node")

// ErrUnexpectedAnswer: the answer matches no Button.Data and the node takes no free
// text.
var ErrUnexpectedAnswer = errors.New("unexpected answer")

// ErrBadAnswer: Node.Parse refused the answer, so the same node is asked again.
var ErrBadAnswer = errors.New("bad answer")
