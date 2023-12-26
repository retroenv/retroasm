package ast

// Operator ...
type Operator struct {
	*node

	Operator string
}

// NewOperator returns a new operator node.
func NewOperator(operator string) Operator {
	return Operator{
		node:     &node{},
		Operator: operator,
	}
}
