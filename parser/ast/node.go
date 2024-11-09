// Package ast implements abstract syntax tree (AST) types.
package ast

// Node ...
type Node interface {
	// Copy returns a copy of the node. Used for copying nodes in rept support.
	Copy() Node
	// SetComment sets the comment for the node.
	SetComment(message string)
}

type node struct {
	comment Comment
}

// SetComment sets the comment for the node.
func (n *node) SetComment(message string) {
	n.comment.Message = message
}
