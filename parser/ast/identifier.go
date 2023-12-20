package ast

import "github.com/retroenv/assembler/lexer/token"

// Identifier ...
type Identifier struct {
	Name      string
	Arguments []token.Token

	Comment Comment
}

func (i *Identifier) node() {}

// SetComment sets the comment for the node.
func (i *Identifier) SetComment(message string) {
	i.Comment.Message = message
}
