package ast

import (
	"github.com/retroenv/assembler/expression"
)

// Alias ...
type Alias struct {
	Name           string
	Expression     *expression.Expression
	SymbolReusable bool // aliases defined with = can be redefined

	Comment Comment
}

func (a *Alias) node() {}

// SetComment sets the comment for the node.
func (a *Alias) SetComment(message string) {
	a.Comment.Message = message
}
