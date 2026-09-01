package assembler

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	retroarch "github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/arch/cpu/cpu6502"
)

// GenerateInstructionOpcode generates the instruction opcode based on the instruction base opcode,
// its addressing mode and parameters.
func GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	var instructionInfo *cpu6502.Instruction
	if id := cpu6502OpcodeID(ins); id != cpu6502.InvalidOpcodeID {
		instructionInfo = cpu6502.InstructionsByID[id]
	}
	if instructionInfo == nil {
		instructionInfo = cpu6502.Instructions[strings.ToLower(ins.Name())]
	}
	addressing := cpu6502.AddressingMode(ins.Addressing())
	addressingInfo := instructionInfo.Addressing[addressing]
	ins.SetOpcodes([]byte{addressingInfo.Opcode})
	ins.SetSize(int(addressingInfo.Size))

	switch addressing {
	case cpu6502.ImpliedAddressing, cpu6502.AccumulatorAddressing:

	case cpu6502.ImmediateAddressing,
		cpu6502.ZeroPageAddressing, cpu6502.ZeroPageXAddressing, cpu6502.ZeroPageYAddressing,
		cpu6502.IndirectXAddressing, cpu6502.IndirectYAddressing:

		if err := generateByteAddressingOpcode(assigner, ins); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case cpu6502.AbsoluteAddressing, cpu6502.AbsoluteXAddressing, cpu6502.AbsoluteYAddressing,
		cpu6502.IndirectAddressing:

		if err := generateWordAddressingOpcode(assigner, ins); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case cpu6502.RelativeAddressing:
		if err := generateRelativeAddressingOpcode(assigner, ins); err != nil {
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
		// Auto-upgrade ZeroPage,X/Y to Absolute,X/Y when value exceeds byte range.
		addressing := cpu6502.AddressingMode(ins.Addressing())
		upgraded := upgradeToAbsolute(addressing)
		if upgraded != addressing {
			return upgradeAndGenerateWord(ins, upgraded, value)
		}
		return fmt.Errorf("value %d exceeds byte", value)
	}

	opcodes := append(ins.Opcodes(), byte(value))
	ins.SetOpcodes(opcodes)
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
func upgradeAndGenerateWord(ins arch.Instruction, newMode cpu6502.AddressingMode, value uint64) error {
	var instructionInfo *cpu6502.Instruction
	if id := cpu6502OpcodeID(ins); id != cpu6502.InvalidOpcodeID {
		instructionInfo = cpu6502.InstructionsByID[id]
	}
	if instructionInfo == nil {
		instructionInfo = cpu6502.Instructions[strings.ToLower(ins.Name())]
	}
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

func cpu6502OpcodeID(ins arch.Instruction) cpu6502.OpcodeID {
	identity := ins.OpcodeID()
	if !identity.ValidFor(retroarch.CPU6502) || identity.Value > uint16(cpu6502.OpcodeIDMax) {
		return cpu6502.InvalidOpcodeID
	}
	return cpu6502.OpcodeID(identity.Value)
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
