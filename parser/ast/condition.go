package ast

import (
	"github.com/retroenv/retroasm/expression"
	"github.com/retroenv/retroasm/lexer/token"
)

// If ...
type If struct {
	*node

	Condition *expression.Expression
}

// NewIf returns a new if node.
func NewIf(condition []token.Token) If {
	return If{
		node:      &node{},
		Condition: expression.New(condition...),
	}
}

// Ifdef ...
type Ifdef struct {
	*node

	Identifier string
}

// NewIfdef returns a new ifdef node.
func NewIfdef(identifier string) Ifdef {
	return Ifdef{
		node:       &node{},
		Identifier: identifier,
	}
}

// Ifndef ...
type Ifndef struct {
	*node

	Identifier string
}

// NewIfndef returns a new ifndef node.
func NewIfndef(identifier string) Ifndef {
	return Ifndef{
		node:       &node{},
		Identifier: identifier,
	}
}

// Else ...
type Else struct {
	*node
}

// NewElse returns a new else node.
func NewElse() Else {
	return Else{
		node: &node{},
	}
}

// ElseIf ...
type ElseIf struct {
	*node

	Condition *expression.Expression
}

// NewElseIf returns a new elseif node.
func NewElseIf(condition []token.Token) ElseIf {
	return ElseIf{
		node:      &node{},
		Condition: expression.New(condition...),
	}
}

// Endif ...
type Endif struct {
	*node
}

// NewEndif returns a new endif node.
func NewEndif() Endif {
	return Endif{
		node: &node{},
	}
}
