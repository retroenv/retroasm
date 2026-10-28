// Package assembler encodes the supported 8086 instruction forms.
package assembler

import "github.com/retroenv/retroasm/pkg/arch"

// AssignInstructionAddress uses the same encoding selection as opcode generation.
func AssignInstructionAddress(assigner arch.AddressAssigner, instruction arch.Instruction) (uint64, error) {
	instruction.SetAddress(assigner.ProgramCounter())
	encoded, err := encode(assigner, instruction, false)
	if err != nil {
		return 0, err
	}
	instruction.SetSize(len(encoded))

	return instruction.Address() + uint64(len(encoded)), nil
}
