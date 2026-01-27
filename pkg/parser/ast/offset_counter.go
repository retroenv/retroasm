package ast

// OffsetCounter ...
type OffsetCounter struct {
	*node

	Number uint64
}

// NewOffsetCounter returns a new offset number node.
func NewOffsetCounter(value uint64) Number {
	return Number{
		node:  &node{},
		Value: value,
	}
}

// Copy returns a copy of the offset number node.
func (o OffsetCounter) Copy() Node {
	return OffsetCounter{
		node:   o.node,
		Number: o.Number,
	}
}
