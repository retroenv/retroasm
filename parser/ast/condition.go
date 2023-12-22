package ast

import (
	"github.com/retroenv/assembler/expression"
)

// If ...
type If struct {
	node

	Condition *expression.Expression
}

// Ifdef ...
type Ifdef struct {
	node

	Identifier string
}

// Ifndef ...
type Ifndef struct {
	node

	Identifier string
}

// Else ...
type Else struct {
	node
}

// ElseIf ...
type ElseIf struct {
	node

	Condition *expression.Expression
}

// Endif ...
type Endif struct {
	node
}
