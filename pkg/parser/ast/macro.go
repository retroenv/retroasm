package ast

import (
	"slices"

	"github.com/retroenv/retroasm/pkg/lexer/token"
)

// Macro ...
type Macro struct {
	*node

	Name      string
	Arguments []string
	Token     []token.Token
}

// NewMacro returns a new macro node.
func NewMacro(name string) Macro {
	return Macro{
		node: &node{},
		Name: name,
	}
}

// Copy returns a copy of the macro node.
func (m Macro) Copy() Node {
	return Macro{
		node:      m.node,
		Name:      m.Name,
		Arguments: slices.Clone(m.Arguments),
		Token:     slices.Clone(m.Token),
	}
}
