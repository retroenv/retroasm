package ast

// Modifier represents an instruction address modifier (e.g. +1, -2).
type Modifier struct {
	node

	Operator Operator
	Value    string
}
