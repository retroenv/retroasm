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
