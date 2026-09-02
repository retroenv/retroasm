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
	errMissingInstruction      = sm83parser.ErrMissingInstruction
	errOpcodeNotFound          = sm83parser.ErrOpcodeNotFound
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
	return resolved.OpcodeInfo() //nolint:wrapcheck // parser owns resolved SM83 variant selection
}
