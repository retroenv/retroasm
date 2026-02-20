package ast

// Error represents an assembler error directive (.error).
type Error struct {
	*node

	Message string
}

// NewError returns a new error node.
func NewError(message string) Error {
	return Error{
		node:    &node{},
		Message: message,
	}
}

// Copy returns a copy of the error node.
func (e Error) Copy() Node {
	return Error{
		node:    e.node,
		Message: e.Message,
	}
}
