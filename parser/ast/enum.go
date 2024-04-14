package ast

import (
	"github.com/retroenv/retroasm/expression"
	"github.com/retroenv/retroasm/lexer/token"
)

// Enum ...
type Enum struct {
	*node

	Address *expression.Expression
}

// NewEnum returns a new enum node.
func NewEnum(address []token.Token) Enum {
	return Enum{
		node:    &node{},
		Address: expression.New(address...),
	}
}

// EnumEnd ...
type EnumEnd struct {
	*node
}

// NewEnumEnd returns a new enum end node.
func NewEnumEnd() EnumEnd {
	return EnumEnd{
		node: &node{},
	}
}
