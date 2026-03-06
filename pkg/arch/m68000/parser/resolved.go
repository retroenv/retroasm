// Package parser implements M68000 assembly instruction parsing.
package parser

import (
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/m68000"
)

// ResolvedInstruction contains the fully parsed M68000 instruction.
type ResolvedInstruction struct {
	Instruction *m68000.Instruction
	Size        m68000.OperandSize
	SrcEA       *EffectiveAddress
	DstEA       *EffectiveAddress
	Extra       uint16 // condition code, quick value, trap vector, etc.
}

// EffectiveAddress represents a parsed effective address operand.
type EffectiveAddress struct {
	Mode      m68000.AddressingMode
	Register  uint8            // register number (0-7)
	IndexReg  uint8            // index register for indexed modes
	IndexSize m68000.OperandSize // index register size (.W/.L)
	IsAddrReg bool             // index is address reg (vs data reg)
	Value     ast.Node         // immediate/displacement/address value
	RegList   uint16           // MOVEM register list bitmask
}
