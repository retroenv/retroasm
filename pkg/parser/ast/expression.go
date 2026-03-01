package ast

import (
	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
)

// Expression stores an expression value as an AST node.
type Expression struct {
	*node

	Value *expression.Expression
}

// NewExpression returns a new expression node from tokens.
func NewExpression(tokens ...token.Token) Expression {
	return Expression{
		node:  &node{},
		Value: expression.New(tokens...),
	}
}

// Copy returns a copy of the expression node.
func (e Expression) Copy() Node {
	var valueCopy *expression.Expression
	if e.Value != nil {
		valueCopy = e.Value.Copy()
	}

	return Expression{
		node:  e.node,
		Value: valueCopy,
	}
}
