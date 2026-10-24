package assembler

import (
	"errors"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/cpu68000/parser"
	"github.com/retroenv/retrogolib/arch/cpu/cpu68000"
)

func encodeStatusMOVE(assigner arch.AddressAssigner, resolved parser.ResolvedInstruction) ([]byte, error) {
	if resolved.Size != cpu68000.SizeWord {
		return nil, errors.New("status MOVE requires word size")
	}
	operand := resolved.DstEA
	opcode := uint16(0x40c0)
	if resolved.SrcEA.Mode == cpu68000.StatusRegMode {
		if resolved.SrcEA.Register != parser.RegSR || operand.Mode == cpu68000.StatusRegMode {
			return nil, errors.New("original 68000 cannot read CCR with MOVE")
		}
	} else {
		operand = resolved.SrcEA
		opcode = 0x44c0
		if resolved.DstEA.Register == parser.RegSR {
			opcode = 0x46c0
		}
	}

	mode, register := encodeEAField(operand)
	if mode == 1 || resolved.SrcEA.Mode == cpu68000.StatusRegMode &&
		(operand.Mode == cpu68000.ImmediateMode || operand.Mode == cpu68000.PCDisplacementMode || operand.Mode == cpu68000.PCIndexedMode) {

		return nil, errors.New("invalid status MOVE address mode")
	}
	return appendEAExtensionWords(encodeWord(opcode|uint16(mode)<<3|uint16(register)), assigner, operand, cpu68000.SizeWord)
}

func encodeImmediateStatus(assigner arch.AddressAssigner, resolved parser.ResolvedInstruction,
	opcode uint16) ([]byte, error) {

	if opcode != 0 && opcode != 0x0200 && opcode != 0x0a00 {
		return nil, errors.New("only ORI, ANDI, and EORI can use a status destination")
	}
	size := cpu68000.SizeByte
	if resolved.DstEA.Register == parser.RegSR {
		size = cpu68000.SizeWord
		opcode |= 0x0040
	}

	if resolved.Size != size {
		return nil, errors.New("invalid immediate status operand size")
	}
	return appendEAExtensionWords(encodeWord(opcode|0x003c), assigner, resolved.SrcEA, size)
}
