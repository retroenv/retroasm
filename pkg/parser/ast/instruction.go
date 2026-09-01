package ast

import (
	"slices"

	"github.com/retroenv/retrogolib/arch"
)

// OpcodeID is an architecture-scoped instruction mnemonic identity. Value is
// independent of encoded opcode bytes; zero means unset or unknown.
type OpcodeID struct {
	Architecture arch.Architecture
	Value        uint16
}

// NewOpcodeID returns an architecture-scoped opcode identity.
func NewOpcodeID(architecture arch.Architecture, value uint16) OpcodeID {
	return OpcodeID{Architecture: architecture, Value: value}
}

// ValidFor reports whether the identity is set for architecture.
func (id OpcodeID) ValidFor(architecture arch.Architecture) bool {
	return id.Architecture == architecture && id.Value != 0
}

// Instruction represents a CPU instruction with its addressing mode and operand.
type Instruction struct {
	*node

	OpcodeID OpcodeID
	Name     string
	// Addressing can be any single addressing value or the combined defined
	// values of this package, to allow the assembler to decide which addressing
	// to use
	Addressing int
	Argument   Node
	Modifier   []Modifier
}

// NewInstruction returns a new instruction node with an unset opcode identity.
// Architecture codecs populate OpcodeID after resolving the instruction.
func NewInstruction(name string, addressing int, argument Node, modifier []Modifier) Instruction {
	return Instruction{
		node:       &node{},
		Name:       name,
		Addressing: addressing,
		Argument:   argument,
		Modifier:   modifier,
	}
}

// ArgumentSymbolName returns the instruction argument's label or identifier name.
func (i Instruction) ArgumentSymbolName() string {
	return SymbolName(i.Argument)
}

// SetOpcodeID sets the architecture-scoped opcode identifier.
func (i *Instruction) SetOpcodeID(id OpcodeID) {
	i.OpcodeID = id
}

// WithInstructionOpcodeID returns node with an architecture-scoped opcode ID.
// Value-form instructions are copied; pointer-form instructions are updated in place.
func WithInstructionOpcodeID(n Node, id OpcodeID) Node {
	switch instruction := n.(type) {
	case Instruction:
		instruction.OpcodeID = id
		return instruction
	case *Instruction:
		if instruction != nil {
			instruction.OpcodeID = id
		}
	}
	return n
}

// Copy returns a copy of the instruction node.
func (i Instruction) Copy() Node {
	var arg Node
	if i.Argument != nil {
		arg = i.Argument.Copy()
	}
	return Instruction{
		node:       i.node,
		OpcodeID:   i.OpcodeID,
		Name:       i.Name,
		Addressing: i.Addressing,
		Argument:   arg,
		Modifier:   slices.Clone(i.Modifier),
	}
}
