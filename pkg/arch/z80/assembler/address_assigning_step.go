// Package assembler implements Z80 architecture-specific assembler functionality.
package assembler

import (
	"errors"
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	z80parser "github.com/retroenv/retroasm/pkg/arch/z80/parser"
)

var (
	errUnsupportedArgumentType = errors.New("unsupported z80 argument type")
	errMissingInstruction      = z80parser.ErrMissingInstruction
	errOpcodeNotFound          = z80parser.ErrOpcodeNotFound
)

// AssignInstructionAddress assigns address and size information for a resolved Z80 instruction.
func AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	pc := assigner.ProgramCounter()
	ins.SetAddress(pc)

	resolved, err := resolvedInstruction(ins.Argument())
	if err != nil {
		return 0, fmt.Errorf("resolving instruction argument: %w", err)
	}

	opcodeInfo, addressing, err := resolved.OpcodeInfo()
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
