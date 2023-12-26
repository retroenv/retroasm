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
