package assembler

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/cpu65816/parser"
	"github.com/retroenv/retrogolib/arch/cpu/cpu65816"
)

// GenerateInstructionOpcode generates the instruction opcode based on the instruction base opcode,
// its addressing mode and parameters.
//
//nolint:cyclop,funlen // one explicit dispatch case is retained for each encoded addressing-width family
func GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	resolved, err := resolvedInstruction(ins.Argument())
	if err != nil {
		return fmt.Errorf("resolving instruction argument: %w", err)
	}
	addressing := cpu65816.AddressingMode(ins.Addressing())
	addressingInfo, err := resolved.OpcodeInfo(addressing)
	if err != nil {
		return fmt.Errorf("resolving opcode info: %w", err)
	}
	ins.SetOpcodes([]byte{addressingInfo.Opcode})
	size, err := resolved.EncodedSize(addressing)
	if err != nil {
		return fmt.Errorf("resolving instruction size: %w", err)
	}
	ins.SetSize(size)

	switch addressing {
	case cpu65816.ImpliedAddressing, cpu65816.AccumulatorAddressing:

	case cpu65816.ImmediateAddressing:
		if err := generateImmediateOpcode(assigner, ins, resolved); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case cpu65816.DirectPageAddressing, cpu65816.DirectPageIndexedXAddressing, cpu65816.DirectPageIndexedYAddressing,
		cpu65816.DirectPageIndirectAddressing, cpu65816.DirectPageIndexedXIndirectAddressing,
		cpu65816.DirectPageIndirectIndexedYAddressing,
		cpu65816.DirectPageIndirectLongAddressing, cpu65816.DirectPageIndirectLongIndexedYAddressing,
		cpu65816.StackRelativeAddressing, cpu65816.StackRelativeIndirectIndexedYAddressing:

		if err := generateByteAddressingOpcode(assigner, ins, resolved); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case cpu65816.AbsoluteAddressing, cpu65816.AbsoluteIndexedXAddressing, cpu65816.AbsoluteIndexedYAddressing,
		cpu65816.AbsoluteIndirectAddressing, cpu65816.AbsoluteIndexedXIndirectAddressing,
		cpu65816.AbsoluteIndirectLongAddressing:

		if err := generateWordAddressingOpcode(assigner, ins, resolved); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case cpu65816.AbsoluteLongAddressing, cpu65816.AbsoluteLongIndexedXAddressing:

		if err := generateLongAddressingOpcode(assigner, ins, resolved); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case cpu65816.RelativeAddressing:
		if err := generateRelativeAddressingOpcode(assigner, ins, resolved); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case cpu65816.RelativeLongAddressing:
		if err := generateRelativeLongOpcode(assigner, ins, resolved); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case cpu65816.BlockMoveAddressing:
		if err := generateBlockMoveOpcode(assigner, ins, resolved); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	default:
		return fmt.Errorf("unsupported instruction addressing %d", addressing)
	}

	return nil
}

func generateImmediateOpcode(
	assigner arch.AddressAssigner,
	ins arch.Instruction,
	resolved parser.ResolvedInstruction,
) error {

	value, err := resolvedOperandValue(assigner, resolved, 0)
	if err != nil {
		return fmt.Errorf("getting instruction argument: %w", err)
	}
	switch ins.Size() - 1 {
	case 1:
		if value > math.MaxUint8 {
			return fmt.Errorf("value %d exceeds byte", value)
		}
		ins.SetOpcodes(append(ins.Opcodes(), byte(value)))
	case 2:
		if value > math.MaxUint16 {
			return fmt.Errorf("value %d exceeds word", value)
		}
		ins.SetOpcodes(binary.LittleEndian.AppendUint16(ins.Opcodes(), uint16(value)))
	default:
		return fmt.Errorf("unsupported immediate width %d", ins.Size()-1)
	}
	return nil
}

func generateByteAddressingOpcode(
	assigner arch.AddressAssigner,
	ins arch.Instruction,
	resolved parser.ResolvedInstruction,
) error {

	value, err := resolvedOperandValue(assigner, resolved, 0)
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

func generateWordAddressingOpcode(
	assigner arch.AddressAssigner,
	ins arch.Instruction,
	resolved parser.ResolvedInstruction,
) error {

	value, err := resolvedOperandValue(assigner, resolved, 0)
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

func generateLongAddressingOpcode(
	assigner arch.AddressAssigner,
	ins arch.Instruction,
	resolved parser.ResolvedInstruction,
) error {

	value, err := resolvedOperandValue(assigner, resolved, 0)
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

func generateRelativeAddressingOpcode(
	assigner arch.AddressAssigner,
	ins arch.Instruction,
	resolved parser.ResolvedInstruction,
) error {

	value, err := resolvedOperandValue(assigner, resolved, 0)
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

func generateRelativeLongOpcode(
	assigner arch.AddressAssigner,
	ins arch.Instruction,
	resolved parser.ResolvedInstruction,
) error {

	value, err := resolvedOperandValue(assigner, resolved, 0)
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

func generateBlockMoveOpcode(
	assigner arch.AddressAssigner,
	ins arch.Instruction,
	resolved parser.ResolvedInstruction,
) error {

	src, err := resolvedOperandValue(assigner, resolved, 0)
	if err != nil {
		return fmt.Errorf("getting source bank: %w", err)
	}
	dst, err := resolvedOperandValue(assigner, resolved, 1)
	if err != nil {
		return fmt.Errorf("getting destination bank: %w", err)
	}
	if src > math.MaxUint8 || dst > math.MaxUint8 {
		return fmt.Errorf("block-move banks %d,%d exceed byte", src, dst)
	}

	// 65816 encodes MVN/MVP as: opcode, dst_bank, src_bank
	opcodes := append(ins.Opcodes(), byte(dst), byte(src))
	ins.SetOpcodes(opcodes)
	return nil
}
