package ast

import (
	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
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

// Copy returns a copy of the enum node.
func (e Enum) Copy() Node {
	return Enum{
		node:    e.node,
		Address: e.Address.Copy(),
	}
}

// Copy returns a copy of the enum end node.
func (e EnumEnd) Copy() Node {
	return EnumEnd{
		node: e.node,
	}
}
