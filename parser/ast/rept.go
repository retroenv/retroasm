package ast

import (
	"github.com/retroenv/retroasm/expression"
	"github.com/retroenv/retroasm/lexer/token"
)

// Rept ...
type Rept struct {
	*node

	Count *expression.Expression
}

// NewRept returns a new rept node.
func NewRept(count []token.Token) Rept {
	return Rept{
		node:  &node{},
		Count: expression.New(count...),
	}
}

// Endr ...
type Endr struct {
	*node
}

// NewEndr returns a new rept end node.
func NewEndr() Endr {
	return Endr{
		node: &node{},
	}
}
