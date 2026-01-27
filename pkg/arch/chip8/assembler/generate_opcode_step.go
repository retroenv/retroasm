package assembler

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	coreasm "github.com/retroenv/retroasm/pkg/assembler"
	retrochip8 "github.com/retroenv/retrogolib/arch/cpu/chip8"
)

// GenerateInstructionOpcode generates the instruction opcode based on the instruction base opcode,
// its addressing mode and parameters.
// nolint: cyclop
func GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	instructionInfo := retrochip8.Instructions[ins.Name()]
	addressing := retrochip8.Mode(ins.Addressing())
	addressingInfo := instructionInfo.Addressing[addressing]

	// Start with the base opcode value
	opcode := addressingInfo.Value

	// Apply the mask and encode arguments based on addressing mode
	switch addressing {
	case retrochip8.ImpliedAddressing:
		// No arguments to encode (CLS, RET)

	case retrochip8.AbsoluteAddressing:
		// JP addr, CALL addr - encode 12-bit address
		if err := generateAbsoluteAddressingOpcode(assigner, ins, &opcode); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case retrochip8.V0AbsoluteAddressing:
		// JP V0, addr - encode 12-bit address
		if err := generateAbsoluteAddressingOpcode(assigner, ins, &opcode); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case retrochip8.IAbsoluteAddressing:
		// LD I, addr - encode 12-bit address
		if err := generateAbsoluteAddressingOpcode(assigner, ins, &opcode); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case retrochip8.RegisterValueAddressing:
		// LD Vx, byte / ADD Vx, byte / SE Vx, byte / SNE Vx, byte / RND Vx, byte
		// Argument contains (register << 8) | value
		if err := generateRegisterValueOpcode(assigner, ins, &opcode); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case retrochip8.RegisterRegisterAddressing:
		// LD Vx, Vy / ADD Vx, Vy / OR Vx, Vy / AND Vx, Vy / XOR Vx, Vy / SUB Vx, Vy / etc.
		// Argument contains (register1 << 4) | register2
		if err := generateRegisterRegisterOpcode(ins, &opcode); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case retrochip8.RegisterRegisterNibbleAddressing:
		// DRW Vx, Vy, nibble
		// Argument contains (register1 << 8) | (register2 << 4) | nibble
		if err := generateRegisterRegisterNibbleOpcode(assigner, ins, &opcode); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case retrochip8.RegisterDTAddressing, retrochip8.RegisterKAddressing,
		retrochip8.DTRegisterAddressing, retrochip8.STRegisterAddressing,
		retrochip8.FRegisterAddressing, retrochip8.BRegisterAddressing,
		retrochip8.IRegisterAddressing, retrochip8.IIndirectRegisterAddressing,
		retrochip8.RegisterIndirectIAddressing:
		// All single-register addressing modes
		// Argument contains register number
		if err := generateSingleRegisterOpcode(ins, &opcode); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	default:
		return fmt.Errorf("unsupported instruction addressing %d", addressing)
	}

	// Encode opcode as big-endian 16-bit value
	opcodeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(opcodeBytes, opcode)
	ins.SetOpcodes(opcodeBytes)

	return nil
}

func generateAbsoluteAddressingOpcode(assigner arch.AddressAssigner, ins arch.Instruction, opcode *uint16) error {
	value, err := assigner.ArgumentValue(ins.Argument())
	if err != nil {
		return fmt.Errorf("getting instruction argument: %w", err)
	}
	if value > 0xFFF {
		return fmt.Errorf("address %d exceeds 12-bit range", value)
	}

	// Encode 12-bit address in lower 12 bits
	*opcode |= uint16(value & 0xFFF)
	return nil
}

func generateRegisterValueOpcode(assigner arch.AddressAssigner, ins arch.Instruction, opcode *uint16) error {
	switch arg := ins.Argument().(type) {
	case coreasm.RegisterValueArgument:
		value, err := resolveValue(assigner, arg.Value)
		if err != nil {
			return err
		}
		if value > 0xFF {
			return fmt.Errorf("value %d exceeds byte range", value)
		}
		*opcode |= uint16((uint64(arg.Register) << 8) | value)
		return nil
	case uint64:
		// Extract register (upper 4 bits) and value (lower 8 bits)
		register := (arg >> 8) & 0xF
		byteValue := arg & 0xFF
		*opcode |= uint16((register << 8) | byteValue)
		return nil
	default:
		return errors.New("argument is not a register-value argument")
	}
}

func generateRegisterRegisterOpcode(ins arch.Instruction, opcode *uint16) error {
	value, err := getArgumentValue(ins)
	if err != nil {
		return err
	}

	// Extract registers
	register1 := (value >> 4) & 0xF
	register2 := value & 0xF

	// Encode register1 in bits 8-11 and register2 in bits 4-7
	*opcode |= uint16((register1 << 8) | (register2 << 4))
	return nil
}

func generateRegisterRegisterNibbleOpcode(assigner arch.AddressAssigner, ins arch.Instruction, opcode *uint16) error {
	switch arg := ins.Argument().(type) {
	case coreasm.RegisterRegisterValueArgument:
		value, err := resolveValue(assigner, arg.Value)
		if err != nil {
			return err
		}
		if value > 0xF {
			return fmt.Errorf("nibble %d exceeds 4-bit range", value)
		}
		*opcode |= uint16((uint64(arg.Register1) << 8) | (uint64(arg.Register2) << 4) | value)
		return nil
	case uint64:
		register1 := (arg >> 8) & 0xF
		register2 := (arg >> 4) & 0xF
		nibble := arg & 0xF
		*opcode |= uint16((register1 << 8) | (register2 << 4) | nibble)
		return nil
	default:
		return errors.New("argument is not a register-register-nibble argument")
	}
}

func generateSingleRegisterOpcode(ins arch.Instruction, opcode *uint16) error {
	value, err := getArgumentValue(ins)
	if err != nil {
		return err
	}

	if value > 0xF {
		return fmt.Errorf("register %d exceeds 4-bit range", value)
	}

	// Encode register in bits 8-11
	*opcode |= uint16(value << 8)
	return nil
}

func getArgumentValue(ins arch.Instruction) (uint64, error) {
	arg := ins.Argument()
	if arg == nil {
		return 0, errors.New("missing instruction argument")
	}

	if v, ok := arg.(uint64); ok {
		return v, nil
	}

	return 0, errors.New("argument is not a number")
}

func resolveValue(assigner arch.AddressAssigner, arg any) (uint64, error) {
	value, err := assigner.ArgumentValue(arg)
	if err != nil {
		return 0, fmt.Errorf("getting instruction argument: %w", err)
	}
	return value, nil
}
