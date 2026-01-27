package ast

// Modifier ...
type Modifier struct {
	node

	Operator Operator
	Value    string
}
