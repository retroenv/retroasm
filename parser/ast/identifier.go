package ast

import "github.com/retroenv/assembler/lexer/token"

// Identifier ...
type Identifier struct {
	*node

	Name      string
	Arguments []token.Token
}

// NewIdentifier returns a new identifier node.
func NewIdentifier(name string) Identifier {
	return Identifier{
		node: &node{},
		Name: name,
	}
}
