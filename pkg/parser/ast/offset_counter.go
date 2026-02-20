package ast

// OffsetCounter ...
type OffsetCounter struct {
	*node

	Number uint64
}

// NewOffsetCounter returns a new offset counter node.
func NewOffsetCounter(value uint64) OffsetCounter {
	return OffsetCounter{
		node:   &node{},
		Number: value,
	}
}

// Copy returns a copy of the offset number node.
func (o OffsetCounter) Copy() Node {
	return OffsetCounter{
		node:   o.node,
		Number: o.Number,
	}
}
