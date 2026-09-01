// Package parser implements CPU68000 assembly instruction parsing.
package parser

import (
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu68000"
)

// ResolvedInstruction contains the fully parsed CPU68000 instruction.
type ResolvedInstruction struct {
	Instruction *cpu68000.Instruction
	Size        cpu68000.OperandSize
	SrcEA       *EffectiveAddress
	DstEA       *EffectiveAddress
	Extra       uint16 // condition code, quick value, trap vector, etc.
}

// EffectiveAddress represents a parsed effective address operand.
type EffectiveAddress struct {
	Mode      cpu68000.AddressingMode
	Register  uint8                // register number (0-7)
	IndexReg  uint8                // index register for indexed modes
	IndexSize cpu68000.OperandSize // index register size (.W/.L)
	IsAddrReg bool                 // index is address reg (vs data reg)
	Value     ast.Node             // immediate/displacement/address value
	RegList   uint16               // MOVEM register list bitmask
}

// CopyInstructionArgument returns a deep copy suitable for AST duplication.
func (resolved ResolvedInstruction) CopyInstructionArgument() any {
	resolved.SrcEA = copyEffectiveAddress(resolved.SrcEA)
	resolved.DstEA = copyEffectiveAddress(resolved.DstEA)
	return resolved
}

func copyEffectiveAddress(address *EffectiveAddress) *EffectiveAddress {
	if address == nil {
		return nil
	}
	copied := *address
	if address.Value != nil {
		copied.Value = address.Value.Copy()
	}
	return &copied
}
