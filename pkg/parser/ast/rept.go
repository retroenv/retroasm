package ast

import (
	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
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

// Copy returns a copy of the rept node.
func (r Rept) Copy() Node {
	return Rept{
		node:  r.node,
		Count: r.Count.Copy(),
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

// Copy returns a copy of the rept end node.
func (e Endr) Copy() Node {
	return Endr{
		node: e.node,
	}
}
