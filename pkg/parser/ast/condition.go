package ast

import (
	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
)

// If represents a conditional assembly directive (.if).
type If struct {
	*node

	Condition *expression.Expression
}

// Ifdef represents a conditional assembly directive that checks if a symbol is defined.
type Ifdef struct {
	*node

	Identifier string
}

// Ifndef represents a conditional assembly directive that checks if a symbol is not defined.
type Ifndef struct {
	*node

	Identifier string
}

// Else represents the else branch of a conditional assembly block.
type Else struct {
	*node
}

// ElseIf represents an else-if branch of a conditional assembly block.
type ElseIf struct {
	*node

	Condition *expression.Expression
}

// Endif represents the end of a conditional assembly block.
type Endif struct {
	*node
}

// NewIf returns a new if node.
func NewIf(condition []token.Token) If {
	return If{
		node:      &node{},
		Condition: expression.New(condition...),
	}
}

// NewIfdef returns a new ifdef node.
func NewIfdef(identifier string) Ifdef {
	return Ifdef{
		node:       &node{},
		Identifier: identifier,
	}
}

// NewIfndef returns a new ifndef node.
func NewIfndef(identifier string) Ifndef {
	return Ifndef{
		node:       &node{},
		Identifier: identifier,
	}
}

// NewElse returns a new else node.
func NewElse() Else {
	return Else{
		node: &node{},
	}
}

// NewElseIf returns a new elseif node.
func NewElseIf(condition []token.Token) ElseIf {
	return ElseIf{
		node:      &node{},
		Condition: expression.New(condition...),
	}
}

// NewEndif returns a new endif node.
func NewEndif() Endif {
	return Endif{
		node: &node{},
	}
}

// Copy returns a copy of the if node.
func (i If) Copy() Node {
	return If{
		node:      i.node,
		Condition: i.Condition.Copy(),
	}
}

// Copy returns a copy of the ifdef node.
func (i Ifdef) Copy() Node {
	return Ifdef{
		node:       i.node,
		Identifier: i.Identifier,
	}
}

// Copy returns a copy of the ifndef node.
func (i Ifndef) Copy() Node {
	return Ifndef{
		node:       i.node,
		Identifier: i.Identifier,
	}
}

// Copy returns a copy of the else node.
func (e Else) Copy() Node {
	return Else{
		node: e.node,
	}
}

// Copy returns a copy of the elseif node.
func (e ElseIf) Copy() Node {
	return ElseIf{
		node:      e.node,
		Condition: e.Condition.Copy(),
	}
}

// Copy returns a copy of the endif node.
func (e Endif) Copy() Node {
	return Endif{
		node: e.node,
	}
}
