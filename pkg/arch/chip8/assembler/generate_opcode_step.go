package assembler

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/chip8/parser"
	"github.com/retroenv/retrogolib/arch/cpu/chip8"
)

var errUnsupportedArgumentType = errors.New("unsupported CHIP-8 argument type")

// GenerateInstructionOpcode generates one typed CHIP-8 instruction opcode.
func GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	resolved, err := resolvedInstruction(ins.Argument())
	if err != nil {
		return fmt.Errorf("resolving instruction argument: %w", err)
	}
	addressingInfo, err := resolved.OpcodeInfo()
	if err != nil {
		return fmt.Errorf("resolving opcode info: %w", err)
	}
	opcode := addressingInfo.Value
	if err := generateInstructionArgumentOpcode(assigner, resolved, &opcode); err != nil {
		return fmt.Errorf("generating opcode: %w", err)
	}

	opcodeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(opcodeBytes, opcode)
	ins.SetOpcodes(opcodeBytes)
	return nil
}

func resolvedInstruction(argument any) (parser.ResolvedInstruction, error) {
	resolved, ok := argument.(parser.ResolvedInstruction)
	if !ok {
		return parser.ResolvedInstruction{}, fmt.Errorf("%w: %T", errUnsupportedArgumentType, argument)
	}
	return resolved, nil
}

//nolint:cyclop // one explicit dispatch case is retained for every CHIP-8 addressing family
func generateInstructionArgumentOpcode(
	assigner arch.AddressAssigner,
	resolved parser.ResolvedInstruction,
	opcode *uint16,
) error {

	switch resolved.Addressing {
	case chip8.ImpliedAddressing:
		return nil
	case chip8.AbsoluteAddressing:
		return generateValueOpcode(assigner, resolved.Operands, 0, 0xfff, opcode)
	case chip8.V0AbsoluteAddressing, chip8.IAbsoluteAddressing:
		return generateValueOpcode(assigner, resolved.Operands, 1, 0xfff, opcode)
	case chip8.RegisterAddressing:
		return generateRegisterOpcode(resolved.Operands, 0, opcode)
	case chip8.RegisterValueAddressing:
		if err := generateRegisterOpcode(resolved.Operands, 0, opcode); err != nil {
			return err
		}
		if len(resolved.Operands) == 1 {
			return nil
		}
		return generateValueOpcode(assigner, resolved.Operands, 1, 0xff, opcode)
	case chip8.RegisterRegisterAddressing:
		return generateRegisterPairOpcode(resolved.Operands, opcode)
	case chip8.RegisterRegisterNibbleAddressing:
		if err := generateRegisterPairOpcode(resolved.Operands, opcode); err != nil {
			return err
		}
		return generateValueOpcode(assigner, resolved.Operands, 2, 0xf, opcode)
	case chip8.RegisterDTAddressing, chip8.RegisterKAddressing,
		chip8.DTRegisterAddressing, chip8.STRegisterAddressing,
		chip8.FRegisterAddressing, chip8.BRegisterAddressing,
		chip8.IRegisterAddressing, chip8.IIndirectRegisterAddressing,
		chip8.RegisterIndirectIAddressing:
		return generateSpecialRegisterOpcode(resolved.Operands, opcode)
	default:
		return fmt.Errorf("unsupported instruction addressing %d", resolved.Addressing)
	}
}

func generateRegisterOpcode(operands parser.Operands, index int, opcode *uint16) error {
	if index < 0 || index >= len(operands) || operands[index].Kind != parser.OperandRegister {
		return fmt.Errorf("operand %d is not a register", index)
	}
	register := operands[index].Register
	if register > 0xf {
		return fmt.Errorf("register %d exceeds 4-bit range", register)
	}
	*opcode |= uint16(register) << 8
	return nil
}

func generateRegisterPairOpcode(operands parser.Operands, opcode *uint16) error {
	if err := generateRegisterOpcode(operands, 0, opcode); err != nil {
		return err
	}
	if len(operands) < 2 || operands[1].Kind != parser.OperandRegister {
		return errors.New("operand 1 is not a register")
	}
	register := operands[1].Register
	if register > 0xf {
		return fmt.Errorf("register %d exceeds 4-bit range", register)
	}
	*opcode |= uint16(register) << 4
	return nil
}

func generateSpecialRegisterOpcode(operands parser.Operands, opcode *uint16) error {
	for index, operand := range operands {
		if operand.Kind == parser.OperandRegister {
			return generateRegisterOpcode(operands, index, opcode)
		}
	}
	return errors.New("instruction has no register operand")
}

func generateValueOpcode(
	assigner arch.AddressAssigner,
	operands parser.Operands,
	index int,
	maximum uint64,
	opcode *uint16,
) error {

	if index < 0 || index >= len(operands) || operands[index].Value == nil {
		return fmt.Errorf("operand %d has no value", index)
	}
	value, err := assigner.ArgumentValue(operands[index].Value)
	if err != nil {
		return fmt.Errorf("getting instruction argument: %w", err)
	}
	if value > maximum {
		return fmt.Errorf("value %d exceeds 0x%X", value, maximum)
	}
	*opcode |= uint16(value)
	return nil
}
