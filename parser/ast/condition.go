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

// Copy returns a copy of the if node.
func (i If) Copy() Node {
	return If{
		node:      i.node,
		Condition: i.Condition.Copy(),
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

// Copy returns a copy of the ifdef node.
func (i Ifdef) Copy() Node {
	return Ifdef{
		node:       i.node,
		Identifier: i.Identifier,
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

// Copy returns a copy of the ifndef node.
func (i Ifndef) Copy() Node {
	return Ifndef{
		node:       i.node,
		Identifier: i.Identifier,
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

// Copy returns a copy of the else node.
func (e Else) Copy() Node {
	return Else{
		node: e.node,
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

// Copy returns a copy of the elseif node.
func (e ElseIf) Copy() Node {
	return ElseIf{
		node:      e.node,
		Condition: e.Condition.Copy(),
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

// Copy returns a copy of the endif node.
func (e Endif) Copy() Node {
	return Endif{
		node: e.node,
	}
}
