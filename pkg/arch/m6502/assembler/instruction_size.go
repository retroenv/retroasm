package assembler

import "github.com/retroenv/retrogolib/arch/cpu/m6502"

func instructionSize(addressing m6502.AddressingMode, metadataSize byte) int {
	if metadataSize != 0 {
		return int(metadataSize)
	}

	// Some valid opcodes lack size metadata. Infer their encoded width from the
	// addressing form so address assignment and opcode generation stay aligned.
	switch addressing {
	case m6502.ImpliedAddressing, m6502.AccumulatorAddressing:
		return 1
	case m6502.AbsoluteAddressing, m6502.AbsoluteXAddressing, m6502.AbsoluteYAddressing,
		m6502.IndirectAddressing, m6502.AbsoluteXIndirectAddressing,
		m6502.ZeroPageRelativeAddressing:

		return 3
	default:
		return 2
	}
}
