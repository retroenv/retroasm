// Package assembler implements the architecture specific assembler functionality.
package assembler

import (
	"fmt"

	"github.com/retroenv/retroasm/arch"
	retrochip8 "github.com/retroenv/retrogolib/arch/cpu/chip8"
)

// AssignInstructionAddress assigns an address to the instruction and calculates its size.
func AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	pc := assigner.ProgramCounter()
	ins.SetAddress(pc)

	name := ins.Name()
	insDetails, ok := retrochip8.Instructions[name]
	if !ok {
		return 0, fmt.Errorf("unsupported instruction '%s'", name)
	}

	addressing := retrochip8.Mode(ins.Addressing())
	addressingInfo, ok := insDetails.Addressing[addressing]
	if !ok {
		return 0, fmt.Errorf("unsupported instruction '%s' addressing %d", name, addressing)
	}

	// All Chip-8 instructions are 2 bytes (16-bit)
	const instructionSize = 2
	ins.SetSize(instructionSize)

	_ = addressingInfo // Used for validation

	programCounter := pc + instructionSize
	return programCounter, nil
}
