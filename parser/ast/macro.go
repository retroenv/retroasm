package ast

import "github.com/retroenv/assembler/lexer/token"

// Macro ...
type Macro struct {
	*node

	Name      string
	Arguments []string
	Token     []token.Token
}

// NewMacro returns a new macro node.
func NewMacro(name string) Macro {
	return Macro{
		node: &node{},
		Name: name,
	}
}
