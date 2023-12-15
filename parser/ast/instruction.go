package ast

import (
	. "github.com/retroenv/retrogolib/addressing"
)

const (
	XAddressing = AbsoluteXAddressing | ZeroPageXAddressing
	YAddressing = AbsoluteYAddressing | ZeroPageYAddressing
)

// Instruction ...
type Instruction struct {
	Name string
	// Addressing can be any single addressing value or the combined defined
	// values of this package, to allow the assembler to decide which addressing
	// to use
	Addressing Mode
	Argument   Node
	Modifier   []Modifier

	Comment Comment
}

func (i *Instruction) node() {}

// SetComment sets the comment for the node.
func (i *Instruction) SetComment(message string) {
	i.Comment.Message = message
}
