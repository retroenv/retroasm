package ast

import "github.com/retroenv/assembler/lexer/token"

// Macro ...
type Macro struct {
	Name      string
	Arguments []string
	Token     []token.Token

	Comment Comment
}

func (m *Macro) node() {}

// SetComment sets the comment for the node.
func (m *Macro) SetComment(message string) {
	m.Comment.Message = message
}
