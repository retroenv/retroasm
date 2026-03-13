// Package assembler implements SM83 architecture-specific assembler functionality.
package assembler

import (
	"errors"
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	sm83parser "github.com/retroenv/retroasm/pkg/arch/sm83/parser"
	cpusm83 "github.com/retroenv/retrogolib/arch/cpu/sm83"
)

var (
	errUnsupportedArgumentType = errors.New("unsupported sm83 argument type")
	errMissingInstruction      = errors.New("resolved instruction details are missing")
	errOpcodeNotFound          = errors.New("opcode mapping not found")
)

// AssignInstructionAddress assigns address and size information for a resolved SM83 instruction.
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

func resolvedInstruction(argument any) (sm83parser.ResolvedInstruction, error) {
	resolved, ok := argument.(sm83parser.ResolvedInstruction)
	if !ok {
		return sm83parser.ResolvedInstruction{}, fmt.Errorf("%w: %T", errUnsupportedArgumentType, argument)
	}
	return resolved, nil
}

func opcodeInfoForResolvedInstruction(resolved sm83parser.ResolvedInstruction) (cpusm83.OpcodeInfo, cpusm83.AddressingMode, error) {
	if resolved.Instruction == nil {
		return cpusm83.OpcodeInfo{}, cpusm83.NoAddressing, errMissingInstruction
	}

	// CB bit instructions (BIT/RES/SET) use computed opcodes — use base addressing map entry.
	if resolved.Addressing == cpusm83.BitAddressing {
		info, ok := resolved.Instruction.Addressing[cpusm83.RegisterAddressing]
		if ok {
			return info, cpusm83.BitAddressing, nil
		}
	}

	if len(resolved.RegisterParams) > 0 {
		if info, addressing, ok := opcodeInfoFromRegisterOperands(resolved); ok {
			return info, addressing, nil
		}
	}

	if info, addressing, ok := opcodeInfoFromAddressing(resolved); ok {
		return info, addressing, nil
	}

	return cpusm83.OpcodeInfo{}, cpusm83.NoAddressing, errOpcodeNotFound
}

func opcodeInfoFromRegisterOperands(resolved sm83parser.ResolvedInstruction) (cpusm83.OpcodeInfo, cpusm83.AddressingMode, bool) {
	switch len(resolved.RegisterParams) {
	case 1:
		if len(resolved.Instruction.RegisterOpcodes) == 0 {
			return cpusm83.OpcodeInfo{}, cpusm83.NoAddressing, false
		}
		info, ok := resolved.Instruction.RegisterOpcodes[resolved.RegisterParams[0]]
		if !ok {
			return cpusm83.OpcodeInfo{}, cpusm83.NoAddressing, false
		}
		return info, resolveAddressing(resolved), true

	case 2:
		if len(resolved.Instruction.RegisterPairOpcodes) == 0 {
			return cpusm83.OpcodeInfo{}, cpusm83.NoAddressing, false
		}
		key := [2]cpusm83.RegisterParam{resolved.RegisterParams[0], resolved.RegisterParams[1]}
		info, ok := resolved.Instruction.RegisterPairOpcodes[key]
		if !ok {
			return cpusm83.OpcodeInfo{}, cpusm83.NoAddressing, false
		}
		return info, resolveAddressing(resolved), true

	default:
		return cpusm83.OpcodeInfo{}, cpusm83.NoAddressing, false
	}
}

func opcodeInfoFromAddressing(resolved sm83parser.ResolvedInstruction) (cpusm83.OpcodeInfo, cpusm83.AddressingMode, bool) {
	if len(resolved.Instruction.Addressing) == 0 {
		return cpusm83.OpcodeInfo{}, cpusm83.NoAddressing, false
	}

	if resolved.Addressing != cpusm83.NoAddressing {
		info, ok := resolved.Instruction.Addressing[resolved.Addressing]
		if ok {
			return info, resolved.Addressing, true
		}
		return cpusm83.OpcodeInfo{}, cpusm83.NoAddressing, false
	}

	if len(resolved.Instruction.Addressing) == 1 {
		for addressing, info := range resolved.Instruction.Addressing {
			return info, addressing, true
		}
	}

	return cpusm83.OpcodeInfo{}, cpusm83.NoAddressing, false
}

func resolveAddressing(resolved sm83parser.ResolvedInstruction) cpusm83.AddressingMode {
	if resolved.Addressing != cpusm83.NoAddressing {
		return resolved.Addressing
	}

	if len(resolved.Instruction.Addressing) == 1 {
		for addressing := range resolved.Instruction.Addressing {
			return addressing
		}
	}

	return cpusm83.NoAddressing
}
