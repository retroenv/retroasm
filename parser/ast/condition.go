package ast

import (
	"github.com/retroenv/assembler/expression"
)

// If ...
type If struct {
	Condition *expression.Expression

	Comment Comment
}

func (i If) node() {}

// Ifdef ...
type Ifdef struct {
	Identifier string

	Comment Comment
}

func (i Ifdef) node() {}

// Ifndef ...
type Ifndef struct {
	Identifier string

	Comment Comment
}

func (i Ifndef) node() {}

// Else ...
type Else struct {
	Comment Comment
}

func (e Else) node() {}

// ElseIf ...
type ElseIf struct {
	Condition *expression.Expression

	Comment Comment
}

func (e ElseIf) node() {}

// Endif ...
type Endif struct {
	Comment Comment
}

func (e Endif) node() {}
