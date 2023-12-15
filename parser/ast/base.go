package ast

import (
	"github.com/retroenv/assembler/expression"
)

// Base ...
type Base struct {
	Address *expression.Expression

	Comment Comment
}

func (b *Base) node() {}

// SetComment sets the comment for the node.
func (b *Base) SetComment(message string) {
	b.Comment.Message = message
}
