// Package assembler implements Z80 architecture-specific assembler functionality.
package assembler

import (
	"errors"
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	z80parser "github.com/retroenv/retroasm/pkg/arch/z80/parser"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
)

var (
	errUnsupportedArgumentType = errors.New("unsupported z80 argument type")
	errMissingInstruction      = errors.New("resolved instruction details are missing")
	errOpcodeNotFound          = errors.New("opcode mapping not found")
)

// AssignInstructionAddress assigns address and size information for a resolved Z80 instruction.
func AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	pc := assigner.ProgramCounter()
	ins.SetAddress(pc)

	resolved, err := resolvedInstruction(ins.Argument())
	if err != nil {
		return 0, fmt.Errorf("resolving instruction argument: %w", err)
	}

	opcodeInfo, addressing, err := opcodeInfoForResolvedInstruction(resolved)
	if err != nil {
		return 0, fmt.Errorf("resolving opcode info for '%s': %w", ins.Name(), err)
	}

	ins.SetAddressing(int(addressing))
	ins.SetSize(int(opcodeInfo.Size))

	return pc + uint64(opcodeInfo.Size), nil
}

func resolvedInstruction(argument any) (z80parser.ResolvedInstruction, error) {
	resolved, ok := argument.(z80parser.ResolvedInstruction)
	if !ok {
		return z80parser.ResolvedInstruction{}, fmt.Errorf("%w: %T", errUnsupportedArgumentType, argument)
	}
	return resolved, nil
}

func opcodeInfoForResolvedInstruction(resolved z80parser.ResolvedInstruction) (cpuz80.OpcodeInfo, cpuz80.AddressingMode, error) {
	if resolved.Instruction == nil {
		return cpuz80.OpcodeInfo{}, cpuz80.NoAddressing, errMissingInstruction
	}

	if len(resolved.RegisterParams) > 0 {
		if info, addressing, ok := opcodeInfoFromRegisterOperands(resolved); ok {
			return info, addressing, nil
		}
	}

	if info, addressing, ok := opcodeInfoFromAddressing(resolved); ok {
		return info, addressing, nil
	}

	return cpuz80.OpcodeInfo{}, cpuz80.NoAddressing, errOpcodeNotFound
}

func opcodeInfoFromRegisterOperands(resolved z80parser.ResolvedInstruction) (cpuz80.OpcodeInfo, cpuz80.AddressingMode, bool) {
	switch len(resolved.RegisterParams) {
	case 1:
		if len(resolved.Instruction.RegisterOpcodes) == 0 {
			return cpuz80.OpcodeInfo{}, cpuz80.NoAddressing, false
		}
		info, ok := resolved.Instruction.RegisterOpcodes[resolved.RegisterParams[0]]
		if !ok {
			return cpuz80.OpcodeInfo{}, cpuz80.NoAddressing, false
		}
		return info, resolveAddressing(resolved), true

	case 2:
		if len(resolved.Instruction.RegisterPairOpcodes) == 0 {
			return cpuz80.OpcodeInfo{}, cpuz80.NoAddressing, false
		}
		key := [2]cpuz80.RegisterParam{resolved.RegisterParams[0], resolved.RegisterParams[1]}
		info, ok := resolved.Instruction.RegisterPairOpcodes[key]
		if !ok {
			return cpuz80.OpcodeInfo{}, cpuz80.NoAddressing, false
		}
		return info, resolveAddressing(resolved), true

	default:
		return cpuz80.OpcodeInfo{}, cpuz80.NoAddressing, false
	}
}

func opcodeInfoFromAddressing(resolved z80parser.ResolvedInstruction) (cpuz80.OpcodeInfo, cpuz80.AddressingMode, bool) {
	if len(resolved.Instruction.Addressing) == 0 {
		return cpuz80.OpcodeInfo{}, cpuz80.NoAddressing, false
	}

	if resolved.Addressing != cpuz80.NoAddressing {
		info, ok := resolved.Instruction.Addressing[resolved.Addressing]
		if ok {
			return info, resolved.Addressing, true
		}
		return cpuz80.OpcodeInfo{}, cpuz80.NoAddressing, false
	}

	if len(resolved.Instruction.Addressing) == 1 {
		for addressing, info := range resolved.Instruction.Addressing {
			return info, addressing, true
		}
	}

	return cpuz80.OpcodeInfo{}, cpuz80.NoAddressing, false
}

func resolveAddressing(resolved z80parser.ResolvedInstruction) cpuz80.AddressingMode {
	if resolved.Addressing != cpuz80.NoAddressing {
		return resolved.Addressing
	}

	if len(resolved.Instruction.Addressing) == 1 {
		for addressing := range resolved.Instruction.Addressing {
			return addressing
		}
	}

	return cpuz80.NoAddressing
}
