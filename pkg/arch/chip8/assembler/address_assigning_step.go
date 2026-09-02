// Package assembler implements the architecture specific assembler functionality.
package assembler

import (
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
)

// AssignInstructionAddress assigns an address to the instruction and calculates its size.
func AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	pc := assigner.ProgramCounter()
	ins.SetAddress(pc)

	resolved, err := resolvedInstruction(ins.Argument())
	if err != nil {
		return 0, fmt.Errorf("resolving instruction argument: %w", err)
	}
	if _, err := resolved.OpcodeInfo(); err != nil {
		return 0, fmt.Errorf("resolving opcode info: %w", err)
	}
	if ins.Addressing() != int(resolved.Addressing) {
		return 0, fmt.Errorf(
			"instruction addressing %d does not match retained addressing %d",
			ins.Addressing(),
			resolved.Addressing,
		)
	}

	// All Chip-8 instructions are 2 bytes (16-bit)
	const instructionSize = 2
	ins.SetSize(instructionSize)

	programCounter := pc + instructionSize
	return programCounter, nil
}
