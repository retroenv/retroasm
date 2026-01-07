package ast

import (
	"github.com/retroenv/retroasm/pkg/expression"
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

// Copy returns a copy of the alias node.
func (a Alias) Copy() Node {
	return Alias{
		node:           a.node,
		Name:           a.Name,
		Expression:     a.Expression.Copy(),
		SymbolReusable: a.SymbolReusable,
	}
}
