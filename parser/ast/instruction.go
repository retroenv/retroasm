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
	*node

	Name string
	// Addressing can be any single addressing value or the combined defined
	// values of this package, to allow the assembler to decide which addressing
	// to use
	Addressing Mode
	Argument   Node
	Modifier   []Modifier
}

// NewInstruction returns a new instruction node.
func NewInstruction(name string, addressing Mode, argument Node, modifier []Modifier) Instruction {
	return Instruction{
		node:       &node{},
		Name:       name,
		Addressing: addressing,
		Argument:   argument,
		Modifier:   modifier,
	}
}
