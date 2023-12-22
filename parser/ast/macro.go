package ast

import "github.com/retroenv/assembler/lexer/token"

// Macro ...
type Macro struct {
	node

	Name      string
	Arguments []string
	Token     []token.Token
}
