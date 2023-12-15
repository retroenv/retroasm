package ast

// Modifier ...
type Modifier struct {
	Operator Operator
	Value    string
}

func (m Modifier) node() {}
