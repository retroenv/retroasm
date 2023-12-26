package ast

// Error ...
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
