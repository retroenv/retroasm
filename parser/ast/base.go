package ast

import (
	"github.com/retroenv/retroasm/expression"
	"github.com/retroenv/retroasm/lexer/token"
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
