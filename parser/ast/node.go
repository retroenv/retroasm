// Package ast implements abstract syntax tree (AST) types.
package ast

// Node ...
type Node interface {
	astNode()

	SetComment(message string)
}

type node struct {
	comment Comment
}

func (n node) astNode() {}

// SetComment sets the comment for the node.
func (n *node) SetComment(message string) {
	n.comment.Message = message
}
