package ast

import (
	"slices"
)

// Instruction represents a CPU instruction with its addressing mode and operand.
type Instruction struct {
	*node

	Name string
	// Addressing can be any single addressing value or the combined defined
	// values of this package, to allow the assembler to decide which addressing
	// to use
	Addressing int
	Argument   Node
	Modifier   []Modifier
}

// NewInstruction returns a new instruction node.
func NewInstruction(name string, addressing int, argument Node, modifier []Modifier) Instruction {
	return Instruction{
		node:       &node{},
		Name:       name,
		Addressing: addressing,
		Argument:   argument,
		Modifier:   modifier,
	}
}

// Copy returns a copy of the instruction node.
func (i Instruction) Copy() Node {
	return Instruction{
		node:       i.node,
		Name:       i.Name,
		Addressing: i.Addressing,
		Argument:   i.Argument.Copy(),
		Modifier:   slices.Clone(i.Modifier),
	}
}
