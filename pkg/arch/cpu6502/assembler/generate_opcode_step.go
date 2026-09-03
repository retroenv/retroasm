package assembler

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu6502"
)

// GenerateInstructionOpcode generates the instruction opcode based on the instruction base opcode,
// its addressing mode and parameters.
func GenerateInstructionOpcode(
	assigner arch.AddressAssigner,
	ins arch.Instruction,
	instructionInfo *cpu6502.Instruction,
) error {

	if instructionInfo == nil {
		return fmt.Errorf("unsupported instruction %q", ins.Name())
	}
	addressing := cpu6502.AddressingMode(ins.Addressing())
	addressingInfo := instructionInfo.Addressing[addressing]
	ins.SetOpcodes([]byte{addressingInfo.Opcode})
	ins.SetSize(int(addressingInfo.Size))

	switch addressing {
	case cpu6502.ImpliedAddressing, cpu6502.AccumulatorAddressing:

	case cpu6502.ImmediateAddressing,
		cpu6502.ZeroPageAddressing, cpu6502.ZeroPageXAddressing, cpu6502.ZeroPageYAddressing,
		cpu6502.IndirectXAddressing, cpu6502.IndirectYAddressing,
		cpu6502.ZeroPageIndirectAddressing:

		if err := generateByteAddressingOpcode(assigner, ins, instructionInfo); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case cpu6502.AbsoluteAddressing, cpu6502.AbsoluteXAddressing, cpu6502.AbsoluteYAddressing,
		cpu6502.IndirectAddressing, cpu6502.AbsoluteXIndirectAddressing:

		if err := generateWordAddressingOpcode(assigner, ins); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case cpu6502.RelativeAddressing:
		if err := generateRelativeAddressingOpcode(assigner, ins); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case cpu6502.ZeroPageRelativeAddressing:
		if err := generateZeroPageRelativeAddressingOpcode(assigner, ins); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	default:
		return fmt.Errorf("unsupported instruction addressing %d", addressing)
	}

	return nil
}

func generateByteAddressingOpcode(
	assigner arch.AddressAssigner,
	ins arch.Instruction,
	instructionInfo *cpu6502.Instruction,
) error {

	value, err := assigner.ArgumentValue(ins.Argument())
	if err != nil {
		return fmt.Errorf("getting instruction argument: %w", err)
	}
	if value > math.MaxUint8 {
		// Auto-upgrade ZeroPage,X/Y to Absolute,X/Y when value exceeds byte range.
		addressing := cpu6502.AddressingMode(ins.Addressing())
		upgraded := upgradeToAbsolute(addressing)
		if upgraded != addressing {
			if err := upgradeAndGenerateWord(ins, instructionInfo, upgraded, value); err != nil {
				return err
			}
			recordCPU6502Relocation(assigner, ins, ins.Argument(), 1, ast.AbsoluteRelocation, ast.WidthWord)
			return nil
		}
		return fmt.Errorf("value %d exceeds byte", value)
	}

	opcodes := append(ins.Opcodes(), byte(value))
	ins.SetOpcodes(opcodes)
	recordCPU6502Relocation(assigner, ins, ins.Argument(), 1, ast.AbsoluteRelocation, ast.WidthByte)
	return nil
}

// upgradeToAbsolute maps ZeroPage indexed modes to their Absolute equivalents.
func upgradeToAbsolute(mode cpu6502.AddressingMode) cpu6502.AddressingMode {
	switch mode {
	case cpu6502.ZeroPageXAddressing:
		return cpu6502.AbsoluteXAddressing
	case cpu6502.ZeroPageYAddressing:
		return cpu6502.AbsoluteYAddressing
	case cpu6502.ZeroPageAddressing:
		return cpu6502.AbsoluteAddressing
	default:
		return mode
	}
}

// upgradeAndGenerateWord re-encodes the instruction using the absolute addressing variant.
func upgradeAndGenerateWord(
	ins arch.Instruction,
	instructionInfo *cpu6502.Instruction,
	newMode cpu6502.AddressingMode,
	value uint64,
) error {

	if instructionInfo == nil {
		return fmt.Errorf("value %d exceeds byte (no instruction info for upgrade)", value)
	}
	absInfo, ok := instructionInfo.Addressing[newMode]
	if !ok {
		return fmt.Errorf("value %d exceeds byte (no %d addressing for %s)", value, newMode, ins.Name())
	}
	ins.SetAddressing(int(newMode))
	ins.SetOpcodes([]byte{absInfo.Opcode})
	ins.SetSize(int(absInfo.Size))

	if value > math.MaxUint16 {
		return fmt.Errorf("value %d exceeds word", value)
	}
	opcodes := binary.LittleEndian.AppendUint16(ins.Opcodes(), uint16(value))
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
	recordCPU6502Relocation(assigner, ins, ins.Argument(), 1, ast.AbsoluteRelocation, ast.WidthWord)
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
	recordCPU6502Relocation(assigner, ins, ins.Argument(), 1, ast.RelativeRelocation, ast.WidthByte)
	return nil
}

func generateZeroPageRelativeAddressingOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	arguments, ok := ins.Argument().([]any)
	if !ok || len(arguments) != 2 {
		return fmt.Errorf("zero-page-relative instruction requires two operands, got %T", ins.Argument())
	}
	zeroPage, err := assigner.ArgumentValue(arguments[0])
	if err != nil {
		return fmt.Errorf("getting zero-page argument: %w", err)
	}
	if zeroPage > math.MaxUint8 {
		return fmt.Errorf("value %d exceeds byte", zeroPage)
	}
	target, err := assigner.ArgumentValue(arguments[1])
	if err != nil {
		return fmt.Errorf("getting relative target: %w", err)
	}
	addressAfterInstruction := ins.Address() + uint64(ins.Size())
	offset, err := assigner.RelativeOffset(target, addressAfterInstruction)
	if err != nil {
		difference := int64(target) - int64(addressAfterInstruction)
		return fmt.Errorf(
			"branch target 0x%X too far from instruction at 0x%X (offset %d, limit -128..127)",
			target,
			ins.Address(),
			difference,
		)
	}
	ins.SetOpcodes(append(ins.Opcodes(), byte(zeroPage), offset))
	recordCPU6502Relocation(assigner, ins, arguments[0], 1, ast.AbsoluteRelocation, ast.WidthByte)
	recordCPU6502Relocation(assigner, ins, arguments[1], 2, ast.RelativeRelocation, ast.WidthByte)
	return nil
}

func recordCPU6502Relocation(
	assigner arch.AddressAssigner,
	ins arch.Instruction,
	argument any,
	byteOffset uint64,
	kind ast.RelocationKind,
	width ast.DataWidth,
) {

	arch.RecordInstructionRelocation(assigner, ins, argument, arch.RelocationEncoding{
		ByteOffset:    byteOffset,
		Kind:          kind,
		Width:         width,
		ByteOrder:     ast.ByteOrderLittle,
		ReferenceType: ast.FullAddress,
	})
}
