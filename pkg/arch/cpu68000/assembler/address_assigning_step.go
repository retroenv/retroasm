// Package assembler implements CPU68000 architecture-specific assembler functionality.
package assembler

import (
	"errors"
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/cpu68000/parser"
	"github.com/retroenv/retrogolib/arch/cpu/cpu68000"
)

var errUnsupportedArgumentType = errors.New("unsupported cpu68000 argument type")

// AssignInstructionAddress assigns address and size information for a resolved CPU68000 instruction.
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

func resolvedInstruction(argument any) (parser.ResolvedInstruction, error) {
	resolved, ok := argument.(parser.ResolvedInstruction)
	if !ok {
		return parser.ResolvedInstruction{}, fmt.Errorf("%w: %T", errUnsupportedArgumentType, argument)
	}
	return resolved, nil
}

func instructionSize(resolved parser.ResolvedInstruction) int {
	name := resolved.Instruction.Name

	switch name {
	case cpu68000.NOPName, cpu68000.RTSName, cpu68000.RTEName, cpu68000.RTRName,
		cpu68000.RESETName, cpu68000.TRAPVName, cpu68000.ILLEGALName,
		cpu68000.TRAPName, cpu68000.MOVEQName,
		cpu68000.UNLKName, cpu68000.SWAPName, cpu68000.EXTName, cpu68000.EXGName:
		return 2

	case cpu68000.STOPName, cpu68000.LINKName, cpu68000.DBccName, cpu68000.MOVEPName:
		return 4

	case cpu68000.BccName, cpu68000.BRAName, cpu68000.BSRName:
		if resolved.Size == cpu68000.SizeByte {
			return 2
		}
		return 4

	case cpu68000.MOVEMName:
		return instructionSizeMOVEM(resolved)

	default:
		return 2 + eaExtensionSize(resolved.SrcEA, resolved.Size) + eaExtensionSize(resolved.DstEA, resolved.Size)
	}
}

func instructionSizeMOVEM(resolved parser.ResolvedInstruction) int {
	size := 4 // opcode word + register list word
	if resolved.Extra == 0 {
		size += eaExtensionSize(resolved.DstEA, resolved.Size)
	} else {
		size += eaExtensionSize(resolved.SrcEA, resolved.Size)
	}
	return size
}

func eaExtensionSize(ea *parser.EffectiveAddress, opSize cpu68000.OperandSize) int {
	if ea == nil {
		return 0
	}

	switch ea.Mode {
	case cpu68000.DataRegDirectMode, cpu68000.AddrRegDirectMode,
		cpu68000.AddrRegIndirectMode, cpu68000.PostIncrementMode,
		cpu68000.PreDecrementMode, cpu68000.StatusRegMode,
		cpu68000.QuickImmediateMode:
		return 0

	case cpu68000.DisplacementMode, cpu68000.PCDisplacementMode:
		return 2

	case cpu68000.IndexedMode, cpu68000.PCIndexedMode:
		return 2 // extension word with d8 + index info

	case cpu68000.AbsShortMode:
		return 2

	case cpu68000.AbsLongMode:
		return 4

	case cpu68000.ImmediateMode:
		if opSize == cpu68000.SizeLong {
			return 4
		}
		return 2 // byte and word both use a word extension

	default:
		return 0
	}
}
