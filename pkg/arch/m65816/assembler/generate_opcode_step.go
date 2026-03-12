package assembler

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retrogolib/arch/cpu/m65816"
)

// GenerateInstructionOpcode generates the instruction opcode based on the instruction base opcode,
// its addressing mode and parameters.
func GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	instructionInfo := m65816.Instructions[strings.ToLower(ins.Name())]
	addressing := m65816.AddressingMode(ins.Addressing())
	addressingInfo := instructionInfo.Addressing[addressing]
	ins.SetOpcodes([]byte{addressingInfo.Opcode})
	ins.SetSize(int(addressingInfo.BaseSize))

	switch addressing {
	case m65816.ImpliedAddressing, m65816.AccumulatorAddressing:

	case m65816.ImmediateAddressing,
		m65816.DirectPageAddressing, m65816.DirectPageIndexedXAddressing, m65816.DirectPageIndexedYAddressing,
		m65816.DirectPageIndirectAddressing, m65816.DirectPageIndexedXIndirectAddressing,
		m65816.DirectPageIndirectIndexedYAddressing,
		m65816.DirectPageIndirectLongAddressing, m65816.DirectPageIndirectLongIndexedYAddressing,
		m65816.StackRelativeAddressing, m65816.StackRelativeIndirectIndexedYAddressing:

		if err := generateByteAddressingOpcode(assigner, ins); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case m65816.AbsoluteAddressing, m65816.AbsoluteIndexedXAddressing, m65816.AbsoluteIndexedYAddressing,
		m65816.AbsoluteIndirectAddressing, m65816.AbsoluteIndexedXIndirectAddressing:

		if err := generateWordAddressingOpcode(assigner, ins); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case m65816.AbsoluteLongAddressing, m65816.AbsoluteLongIndexedXAddressing,
		m65816.AbsoluteIndirectLongAddressing:

		if err := generateLongAddressingOpcode(assigner, ins); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case m65816.RelativeAddressing:
		if err := generateRelativeAddressingOpcode(assigner, ins); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case m65816.RelativeLongAddressing:
		if err := generateRelativeLongOpcode(assigner, ins); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case m65816.BlockMoveAddressing:
		if err := generateBlockMoveOpcode(assigner, ins); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	default:
		return fmt.Errorf("unsupported instruction addressing %d", addressing)
	}

	return nil
}

func generateByteAddressingOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	value, err := assigner.ArgumentValue(ins.Argument())
	if err != nil {
		return fmt.Errorf("getting instruction argument: %w", err)
	}
	if value > math.MaxUint8 {
		return fmt.Errorf("value %d exceeds byte", value)
	}

	opcodes := append(ins.Opcodes(), byte(value))
	ins.SetOpcodes(opcodes)
	return nil
}

func generateWordAddressingOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	value, err := assigner.ArgumentValue(ins.Argument())
	if err != nil {
		return fmt.Errorf("getting instruction argument: %w", err)
	}
	if value > math.MaxUint16 {
		return fmt.Errorf("value %d exceeds word", value)
	}

	opcodes := binary.LittleEndian.AppendUint16(ins.Opcodes(), uint16(value))
	ins.SetOpcodes(opcodes)
	return nil
}

func generateLongAddressingOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	value, err := assigner.ArgumentValue(ins.Argument())
	if err != nil {
		return fmt.Errorf("getting instruction argument: %w", err)
	}
	if value > 0xFFFFFF {
		return fmt.Errorf("value %d exceeds 24-bit address", value)
	}

	opcodes := binary.LittleEndian.AppendUint16(ins.Opcodes(), uint16(value&0xFFFF))
	opcodes = append(opcodes, byte(value>>16))
	ins.SetOpcodes(opcodes)
	return nil
}

func generateRelativeAddressingOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	value, err := assigner.ArgumentValue(ins.Argument())
	if err != nil {
		return fmt.Errorf("getting instruction argument: %w", err)
	}

	insAddr := ins.Address() + uint64(ins.Size())
	b, err := assigner.RelativeOffset(value, insAddr)
	if err != nil {
		diff := int64(value) - int64(insAddr)
		return fmt.Errorf("branch target 0x%X too far from instruction at 0x%X (offset %d, limit -128..127)", value, ins.Address(), diff)
	}

	opcodes := append(ins.Opcodes(), b)
	ins.SetOpcodes(opcodes)
	return nil
}

func generateRelativeLongOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	value, err := assigner.ArgumentValue(ins.Argument())
	if err != nil {
		return fmt.Errorf("getting instruction argument: %w", err)
	}

	insAddr := ins.Address() + uint64(ins.Size())
	offset := int64(value) - int64(insAddr)

	if offset < math.MinInt16 || offset > math.MaxInt16 {
		return fmt.Errorf("branch long target 0x%X too far from instruction at 0x%X (offset %d, limit -32768..32767)",
			value, ins.Address(), offset)
	}

	opcodes := binary.LittleEndian.AppendUint16(ins.Opcodes(), uint16(int16(offset)))
	ins.SetOpcodes(opcodes)
	return nil
}

func generateBlockMoveOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	value, err := assigner.ArgumentValue(ins.Argument())
	if err != nil {
		return fmt.Errorf("getting instruction argument: %w", err)
	}

	// Packed as (src << 8) | dst during parsing
	dst := byte(value & 0xFF)
	src := byte((value >> 8) & 0xFF)

	// 65816 encodes MVN/MVP as: opcode, dst_bank, src_bank
	opcodes := append(ins.Opcodes(), dst, src)
	ins.SetOpcodes(opcodes)
	return nil
}
