package assembler

import "github.com/retroenv/retrogolib/arch/cpu/cpu6502"

func instructionSize(addressing cpu6502.AddressingMode, metadataSize byte) int {
	if metadataSize != 0 {
		return int(metadataSize)
	}

	// Some valid opcodes lack size metadata. Infer their encoded width from the
	// addressing form so address assignment and opcode generation stay aligned.
	switch addressing {
	case cpu6502.ImpliedAddressing, cpu6502.AccumulatorAddressing:
		return 1
	case cpu6502.AbsoluteAddressing, cpu6502.AbsoluteXAddressing, cpu6502.AbsoluteYAddressing,
		cpu6502.IndirectAddressing, cpu6502.AbsoluteXIndirectAddressing,
		cpu6502.ZeroPageRelativeAddressing:

		return 3
	default:
		return 2
	}
}
