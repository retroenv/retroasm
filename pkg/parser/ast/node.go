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

// FillLabelIndices clears and fills dest with label names and node indices.
func FillLabelIndices(nodes []Node, dest map[string]int) {
	clear(dest)
	for index, n := range nodes {
		if name, ok := LabelName(n); ok {
			dest[name] = index
		}
	}
}

// InstructionFromNode returns the instruction stored in node.
func InstructionFromNode(n Node) (Instruction, bool) {
	switch instr := n.(type) {
	case Instruction:
		return instr, true
	case *Instruction:
		if instr != nil {
			return *instr, true
		}
	}
	return Instruction{}, false
}

// IsInstruction reports whether node stores a non-nil instruction.
func IsInstruction(n Node) bool {
	switch instr := n.(type) {
	case Instruction:
		return true
	case *Instruction:
		return instr != nil
	default:
		return false
	}
}

// IsLabel reports whether node stores a non-nil label.
func IsLabel(n Node) bool {
	switch label := n.(type) {
	case Label:
		return true
	case *Label:
		return label != nil
	default:
		return false
	}
}

// IdentifierName returns the name stored in an identifier node.
func IdentifierName(n Node) (string, bool) {
	switch identifier := n.(type) {
	case Identifier:
		return identifier.Name, true
	case *Identifier:
		if identifier != nil {
			return identifier.Name, true
		}
	}
	return "", false
}

// LabelIndices returns label names mapped to their node indices.
func LabelIndices(nodes []Node) map[string]int {
	indices := make(map[string]int)
	FillLabelIndices(nodes, indices)
	return indices
}

// LabelName returns the name stored in a label node.
func LabelName(n Node) (string, bool) {
	switch label := n.(type) {
	case Label:
		return label.Name, true
	case *Label:
		if label != nil {
			return label.Name, true
		}
	}
	return "", false
}

// NumberValue returns the value stored in a number node.
func NumberValue(n Node) (uint64, bool) {
	switch number := n.(type) {
	case Number:
		return number.Value, true
	case *Number:
		if number != nil {
			return number.Value, true
		}
	}
	return 0, false
}

// SameOperand reports whether two number or symbol nodes represent the same operand.
func SameOperand(a, b Node) bool {
	if a == nil || b == nil {
		return a == b
	}
	if av, ok := NumberValue(a); ok {
		bv, ok := NumberValue(b)
		return ok && av == bv
	}
	av, ok := symbolName(a)
	if !ok {
		return false
	}
	bv, ok := symbolName(b)
	return ok && av == bv
}

// SymbolName returns the name stored in a label or identifier node.
func SymbolName(n Node) string {
	name, _ := symbolName(n)
	return name
}

func symbolName(n Node) (string, bool) {
	if name, ok := LabelName(n); ok {
		return name, true
	}
	return IdentifierName(n)
}
