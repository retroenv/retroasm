package ast

import (
	"slices"
)

// OpcodeIDLookup is an optional architecture-specific resolver that maps a
// lowercase mnemonic name to a numeric OpcodeID. When set, NewInstruction
// calls it automatically so every created instruction has OpcodeID populated.
// Set this once at program startup for the target architecture.
var OpcodeIDLookup func(name string) uint8

// Instruction represents a CPU instruction with its addressing mode and operand.
type Instruction struct {
	*node

	OpcodeID uint8 // Architecture-defined numeric opcode identifier; 0 = unset/unknown
	Name     string
	// Addressing can be any single addressing value or the combined defined
	// values of this package, to allow the assembler to decide which addressing
	// to use
	Addressing int
	Argument   Node
	Modifier   []Modifier
}

// NewInstruction returns a new instruction node. If OpcodeIDLookup is
// registered, OpcodeID is populated automatically from the instruction name.
func NewInstruction(name string, addressing int, argument Node, modifier []Modifier) Instruction {
	var opcodeID uint8
	if OpcodeIDLookup != nil {
		opcodeID = OpcodeIDLookup(name)
	}
	return Instruction{
		node:       &node{},
		OpcodeID:   opcodeID,
		Name:       name,
		Addressing: addressing,
		Argument:   argument,
		Modifier:   modifier,
	}
}

// SetOpcodeID sets the architecture-defined numeric opcode identifier for fast O(1) lookup.
func (i *Instruction) SetOpcodeID(id uint8) {
	i.OpcodeID = id
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
