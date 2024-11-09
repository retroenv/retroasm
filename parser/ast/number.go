package ast

// Number ...
type Number struct {
	*node

	Value uint64
}

// NewNumber returns a new number node.
func NewNumber(value uint64) Number {
	return Number{
		node:  &node{},
		Value: value,
	}
}

// Copy returns a copy of the number node.
func (n Number) Copy() Node {
	return Number{
		node:  n.node,
		Value: n.Value,
	}
}
