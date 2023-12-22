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
	node

	Name string
	// Addressing can be any single addressing value or the combined defined
	// values of this package, to allow the assembler to decide which addressing
	// to use
	Addressing Mode
	Argument   Node
	Modifier   []Modifier
}
