package ast

import (
	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
)

// Base ...
type Base struct {
	*node

	Address *expression.Expression
}

// NewBase returns a new base node.
func NewBase(addressTokens []token.Token) Base {
	return Base{
		node:    &node{},
		Address: expression.New(addressTokens...),
	}
}

// Copy returns a copy of the base node.
func (b Base) Copy() Node {
	return Base{
		node:    b.node,
		Address: b.Address.Copy(),
	}
}
