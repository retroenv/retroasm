// Package assembler implements M68000 architecture-specific assembler functionality.
package assembler

import (
	"errors"
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	m68000parser "github.com/retroenv/retroasm/pkg/arch/m68000/parser"
	"github.com/retroenv/retrogolib/arch/cpu/m68000"
)

var errUnsupportedArgumentType = errors.New("unsupported m68000 argument type")

// AssignInstructionAddress assigns address and size information for a resolved M68000 instruction.
func AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	pc := assigner.ProgramCounter()
	ins.SetAddress(pc)

	resolved, err := resolvedInstruction(ins.Argument())
	if err != nil {
		return 0, fmt.Errorf("resolving instruction argument: %w", err)
	}

	size := instructionSize(resolved)
	ins.SetSize(size)

	return pc + uint64(size), nil
}

func resolvedInstruction(argument any) (m68000parser.ResolvedInstruction, error) {
	resolved, ok := argument.(m68000parser.ResolvedInstruction)
	if !ok {
		return m68000parser.ResolvedInstruction{}, fmt.Errorf("%w: %T", errUnsupportedArgumentType, argument)
	}
	return resolved, nil
}

func instructionSize(resolved m68000parser.ResolvedInstruction) int {
	size := 2 // base opcode word

	name := resolved.Instruction.Name

	// Special cases
	switch name {
	case m68000.NOPName, m68000.RTSName, m68000.RTEName, m68000.RTRName,
		m68000.RESETName, m68000.TRAPVName, m68000.ILLEGALName:
		return 2
	case m68000.TRAPName:
		return 2 // vector encoded in opcode
	case m68000.MOVEQName:
		return 2 // immediate encoded in opcode word
	case m68000.STOPName:
		return 4 // opcode + immediate word
	case m68000.LINKName:
		return 4 // opcode + displacement word
	case m68000.UNLKName, m68000.SWAPName:
		return 2
	case m68000.EXTName:
		return 2
	case m68000.EXGName:
		return 2
	}

	// Branch instructions
	if name == m68000.BccName || name == m68000.BRAName || name == m68000.BSRName {
		if resolved.Size == m68000.SizeByte {
			return 2 // 8-bit displacement in opcode
		}
		return 4 // 16-bit displacement word
	}

	// DBcc: opcode + displacement word
	if name == m68000.DBccName {
		return 4
	}

	// MOVEM: opcode + register list word + EA extension
	if name == m68000.MOVEMName {
		size += 2 // register list word
		if resolved.Extra == 0 {
			// register-to-memory: DstEA has extension words
			size += eaExtensionSize(resolved.DstEA, resolved.Size)
		} else {
			// memory-to-register: SrcEA has extension words
			size += eaExtensionSize(resolved.SrcEA, resolved.Size)
		}
		return size
	}

	// MOVEP: opcode + displacement word
	if name == m68000.MOVEPName {
		return 4
	}

	// General case: add extension words for each EA
	size += eaExtensionSize(resolved.SrcEA, resolved.Size)
	size += eaExtensionSize(resolved.DstEA, resolved.Size)

	// Immediate source for ALU immediate instructions
	if isImmediateALU(name) && resolved.SrcEA == nil {
		// The immediate value is part of SrcEA, already counted
	}

	return size
}

func eaExtensionSize(ea *m68000parser.EffectiveAddress, opSize m68000.OperandSize) int {
	if ea == nil {
		return 0
	}

	switch ea.Mode {
	case m68000.DataRegDirectMode, m68000.AddrRegDirectMode,
		m68000.AddrRegIndirectMode, m68000.PostIncrementMode,
		m68000.PreDecrementMode, m68000.StatusRegMode,
		m68000.QuickImmediateMode:
		return 0

	case m68000.DisplacementMode, m68000.PCDisplacementMode:
		return 2

	case m68000.IndexedMode, m68000.PCIndexedMode:
		return 2 // extension word with d8 + index info

	case m68000.AbsShortMode:
		return 2

	case m68000.AbsLongMode:
		return 4

	case m68000.ImmediateMode:
		if opSize == m68000.SizeLong {
			return 4
		}
		return 2 // byte and word both use a word extension

	default:
		return 0
	}
}

func isImmediateALU(name string) bool {
	switch name {
	case m68000.ADDIName, m68000.SUBIName, m68000.ANDIName,
		m68000.ORIName, m68000.EORIName, m68000.CMPIName:
		return true
	}
	return false
}
