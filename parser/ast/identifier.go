package ast

import (
	"slices"

	"github.com/retroenv/retroasm/lexer/token"
)

// Identifier ...
type Identifier struct {
	*node

	Name      string
	Arguments []token.Token
}

// NewIdentifier returns a new identifier node.
func NewIdentifier(name string) Identifier {
	return Identifier{
		node: &node{},
		Name: name,
	}
}

// Copy returns a copy of the identifier node.
func (i Identifier) Copy() Node {
	return Identifier{
		node:      i.node,
		Name:      i.Name,
		Arguments: slices.Clone(i.Arguments),
	}
}
