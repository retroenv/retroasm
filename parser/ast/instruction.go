package ast

import (
	"slices"

	"github.com/retroenv/retrogolib/arch/cpu/m6502"
)

const (
	XAddressing = m6502.AbsoluteXAddressing | m6502.ZeroPageXAddressing
	YAddressing = m6502.AbsoluteYAddressing | m6502.ZeroPageYAddressing
)

// Instruction ...
type Instruction struct {
	*node

	Name string
	// Addressing can be any single addressing value or the combined defined
	// values of this package, to allow the assembler to decide which addressing
	// to use
	Addressing m6502.AddressingMode
	Argument   Node
	Modifier   []Modifier
}

// NewInstruction returns a new instruction node.
func NewInstruction(name string, addressing m6502.AddressingMode, argument Node, modifier []Modifier) Instruction {
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
