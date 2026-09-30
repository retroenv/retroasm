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

// NewOpcodeID returns an architecture-scoped opcode identity.
func NewOpcodeID(architecture arch.Architecture, value uint16) OpcodeID {
	return OpcodeID{
		Architecture: architecture,
		Value:        value,
	}
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

// ValidFor reports whether the identity is set for architecture.
func (id OpcodeID) ValidFor(architecture arch.Architecture) bool {
	return id.Architecture == architecture && id.Value != 0
}

// ArgumentSymbolName returns the instruction argument's label or identifier name.
func (i Instruction) ArgumentSymbolName() string {
	return SymbolName(i.Argument)
}

// SetOpcodeID sets the architecture-scoped opcode identifier.
func (i *Instruction) SetOpcodeID(id OpcodeID) {
	i.OpcodeID = id
}

// Copy returns a copy of the instruction node.
func (i Instruction) Copy() Node {
	metadata, arg := copyInstructionArgumentAndMetadata(i)
	return Instruction{
		node:       metadata,
		OpcodeID:   i.OpcodeID,
		Name:       i.Name,
		Addressing: i.Addressing,
		Argument:   arg,
		Modifier:   copyInstructionModifiers(i.Modifier),
	}
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

func copyInstructionArgumentAndMetadata(i Instruction) (*node, Node) {
	switch argument := i.Argument.(type) {
	case Number:
		metadata, operandMetadata := copyInstructionMetadata(i.node, argument.node)
		argument.node = operandMetadata
		return metadata, argument
	case Identifier:
		metadata, operandMetadata := copyInstructionMetadata(i.node, argument.node)
		argument.node = operandMetadata
		argument.Arguments = slices.Clone(argument.Arguments)
		return metadata, argument
	default:
		var argumentCopy Node
		if i.Argument != nil {
			argumentCopy = i.Argument.Copy()
		}
		return i.node.copyNode(), argumentCopy
	}
}

// The instruction and operand have separate metadata in one allocation.
func copyInstructionMetadata(instruction, operand *node) (*node, *node) {
	metadata := &[2]node{}
	if instruction != nil {
		metadata[0] = *instruction
	}
	if operand != nil {
		metadata[1] = *operand
	}
	return &metadata[0], &metadata[1]
}

func copyInstructionModifiers(modifiers []Modifier) []Modifier {
	copied := slices.Clone(modifiers)
	for index := range copied {
		if copied[index].Operator.node != nil {
			copied[index].Operator.node = copied[index].Operator.node.copyNode()
		}
	}
	return copied
}
