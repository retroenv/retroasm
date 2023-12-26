package ast

import (
	"github.com/retroenv/assembler/expression"
)

// Alias ...
type Alias struct {
	*node

	Name           string
	Expression     *expression.Expression
	SymbolReusable bool // aliases defined with = can be redefined
}

// NewAlias returns a new alias node.
func NewAlias(name string) Alias {
	return Alias{
		node:       &node{},
		Name:       name,
		Expression: &expression.Expression{},
	}
}
