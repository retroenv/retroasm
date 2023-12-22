package ast

import "github.com/retroenv/assembler/lexer/token"

// Identifier ...
type Identifier struct {
	node

	Name      string
	Arguments []token.Token
}
