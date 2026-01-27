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

// Copy returns a copy of the operator node.
func (o Operator) Copy() Node {
	return Operator{
		node:     o.node,
		Operator: o.Operator,
	}
}
