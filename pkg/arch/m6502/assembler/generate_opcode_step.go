package assembler

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retrogolib/arch/cpu/m6502"
)

// GenerateInstructionOpcode generates the instruction opcode based on the instruction base opcode,
// its addressing mode and parameters.
func GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	instructionInfo := m6502.Instructions[strings.ToLower(ins.Name())]
	addressing := m6502.AddressingMode(ins.Addressing())
	addressingInfo := instructionInfo.Addressing[addressing]
	ins.SetOpcodes([]byte{addressingInfo.Opcode})
	ins.SetSize(int(addressingInfo.Size))

	switch addressing {
	case m6502.ImpliedAddressing, m6502.AccumulatorAddressing:

	case m6502.ImmediateAddressing:
		if err := generateImmediateAddressingOpcode(assigner, ins); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case m6502.AbsoluteAddressing, m6502.AbsoluteXAddressing, m6502.AbsoluteYAddressing,
		m6502.IndirectAddressing, m6502.IndirectXAddressing, m6502.IndirectYAddressing:
		if err := generateAbsoluteIndirectAddressingOpcode(assigner, ins); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case m6502.ZeroPageAddressing, m6502.ZeroPageXAddressing, m6502.ZeroPageYAddressing:
		if err := generateZeroPageAddressingOpcode(assigner, ins); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case m6502.RelativeAddressing:
		if err := generateRelativeAddressingOpcode(assigner, ins); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	default:
		return fmt.Errorf("unsupported instruction addressing %d", addressing)
	}

	return nil
}

func generateAbsoluteIndirectAddressingOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
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

func generateZeroPageAddressingOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
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

func generateImmediateAddressingOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
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
