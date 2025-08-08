// Package ast implements abstract syntax tree (AST) types for assembly language parsing.
//
// The AST represents the parsed structure of assembly code and includes:
//   - Instructions with addressing modes and arguments
//   - Data directives (bytes, words, strings)
//   - Control flow (labels, conditional assembly)
//   - Macro definitions and expansions
//   - Configuration and include directives
//
// All AST nodes implement the Node interface, which supports copying for
// macro expansion and comment attachment for documentation.
package ast

// Node represents a single element in the assembly language AST.
//
// All AST nodes must support deep copying for macro expansion (.rept directives)
// and comment attachment for inline documentation.
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
