package ast

// Bank ...
type Bank struct {
	*node

	Number int
}

// NewBank returns a new bank node.
func NewBank(number int) Bank {
	return Bank{
		node:   &node{},
		Number: number,
	}
}

// Copy returns a copy of the bank node.
func (b Bank) Copy() Node {
	return Bank{
		node:   b.node,
		Number: b.Number,
	}
}
