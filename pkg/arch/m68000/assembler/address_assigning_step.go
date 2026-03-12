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
	name := resolved.Instruction.Name

	switch name {
	case m68000.NOPName, m68000.RTSName, m68000.RTEName, m68000.RTRName,
		m68000.RESETName, m68000.TRAPVName, m68000.ILLEGALName,
		m68000.TRAPName, m68000.MOVEQName,
		m68000.UNLKName, m68000.SWAPName, m68000.EXTName, m68000.EXGName:
		return 2

	case m68000.STOPName, m68000.LINKName, m68000.DBccName, m68000.MOVEPName:
		return 4

	case m68000.BccName, m68000.BRAName, m68000.BSRName:
		if resolved.Size == m68000.SizeByte {
			return 2
		}
		return 4

	case m68000.MOVEMName:
		return instructionSizeMOVEM(resolved)

	default:
		return 2 + eaExtensionSize(resolved.SrcEA, resolved.Size) + eaExtensionSize(resolved.DstEA, resolved.Size)
	}
}

func instructionSizeMOVEM(resolved m68000parser.ResolvedInstruction) int {
	size := 4 // opcode word + register list word
	if resolved.Extra == 0 {
		size += eaExtensionSize(resolved.DstEA, resolved.Size)
	} else {
		size += eaExtensionSize(resolved.SrcEA, resolved.Size)
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
