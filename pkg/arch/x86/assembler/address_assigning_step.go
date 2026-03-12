// Package assembler implements the x86 architecture specific assembler functionality.
package assembler

import (
	"github.com/retroenv/retroasm/pkg/arch"
)

// AssignInstructionAddress assigns an address to an x86 instruction and calculates its size.
func AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	pc := assigner.ProgramCounter()
	ins.SetAddress(pc)

	// For x86, instruction sizes are determined by addressing mode
	var size int
	switch ins.Addressing() {
	case 0: // RegisterAddressing
		size = getRegisterInstructionSize(ins.Name())
	case 1: // ImmediateAddressing
		size = getImmediateInstructionSize(ins.Name())
	case 2: // DirectAddressing
		size = 4 // opcode + modrm + 16-bit address
	default:
		size = 2 // default
	}

	ins.SetSize(size)
	programCounter := pc + uint64(size)
	return programCounter, nil
}

func getRegisterInstructionSize(name string) int {
	switch name {
	case "NOP", "RET":
		return 1
	case "INC", "DEC", "PUSH", "POP":
		return 1
	case "MOV", "ADD", "SUB", "CMP", "AND", "OR", "XOR":
		return 2 // opcode + modrm
	default:
		return 2
	}
}

func getImmediateInstructionSize(name string) int {
	switch name {
	case "JE", "JNE", "JC", "JNC", "JL", "JLE", "JG", "JGE", "JS", "JNS", "JO", "JNO":
		return 2 // opcode + 8-bit relative
	case "JMP", "CALL":
		return 3 // opcode + 16-bit relative
	case "MOV", "ADD", "SUB", "CMP", "AND", "OR", "XOR":
		return 3 // opcode + 16-bit immediate
	default:
		return 3
	}
}
