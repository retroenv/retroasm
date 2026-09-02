package parser

import (
	"errors"
	"fmt"
	"math"

	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/chip8"
)

var (
	// ErrInvalidInstruction indicates inconsistent typed CHIP-8 instruction state.
	ErrInvalidInstruction = errors.New("invalid typed CHIP-8 instruction")
	// ErrUnsupportedAddressing indicates an operand shape unavailable for a mnemonic.
	ErrUnsupportedAddressing = errors.New("unsupported CHIP-8 addressing")
)

// ResolvedInstruction retains the selected CHIP-8 instruction and source operands.
type ResolvedInstruction struct {
	Instruction *chip8.Instruction
	Addressing  chip8.Mode
	Operands    Operands
}

// CopyInstructionArgument returns a deep copy suitable for AST duplication.
func (resolved ResolvedInstruction) CopyInstructionArgument() any {
	resolved.Operands = copyOperands(resolved.Operands)
	return resolved
}

// OpcodeInfo returns encoding metadata for the retained addressing mode.
func (resolved ResolvedInstruction) OpcodeInfo() (chip8.OpcodeInfo, error) {
	if resolved.Instruction == nil {
		return chip8.OpcodeInfo{}, fmt.Errorf("%w: missing instruction metadata", ErrInvalidInstruction)
	}
	info, ok := resolved.Instruction.Addressing[resolved.Addressing]
	if !ok {
		return chip8.OpcodeInfo{}, fmt.Errorf(
			"%w: %s mode %d",
			ErrUnsupportedAddressing,
			resolved.Instruction.Name,
			resolved.Addressing,
		)
	}
	return info, nil
}

func resolveOperands(instruction *chip8.Instruction, operands Operands) (chip8.Mode, error) {
	if instruction == nil {
		return chip8.NoAddressing, ErrUnsupportedAddressing
	}
	var addressing chip8.Mode
	switch len(operands) {
	case 0:
		addressing = chip8.ImpliedAddressing
	case 1:
		addressing = resolveSingleOperand(instruction, operands[0])
	case 2:
		addressing = resolveTwoOperands(operands[0], operands[1])
	case 3:
		if operands[0].Kind == OperandRegister && operands[1].Kind == OperandRegister &&
			operands[2].Kind == OperandNibble {

			addressing = chip8.RegisterRegisterNibbleAddressing
		}
	}
	if _, ok := instruction.Addressing[addressing]; !ok {
		return chip8.NoAddressing, ErrUnsupportedAddressing
	}
	return addressing, nil
}

func resolveSingleOperand(instruction *chip8.Instruction, operand Operand) chip8.Mode {
	switch operand.Kind {
	case OperandRegister:
		if _, ok := instruction.Addressing[chip8.RegisterAddressing]; ok {
			return chip8.RegisterAddressing
		}
		if instruction.Name == chip8.SkpName || instruction.Name == chip8.SknpName {
			return chip8.RegisterValueAddressing
		}
	case OperandAddress:
		return chip8.AbsoluteAddressing
	}
	return chip8.NoAddressing
}

func resolveTwoOperands(first, second Operand) chip8.Mode { //nolint:cyclop // complete CHIP-8 operand-pair grammar
	switch {
	case first.Kind == OperandRegister && second.Kind == OperandByte:
		return chip8.RegisterValueAddressing
	case first.Kind == OperandRegister && second.Kind == OperandRegister:
		return chip8.RegisterRegisterAddressing
	case first.Kind == OperandRegister && first.Register == 0 && second.Kind == OperandAddress:
		return chip8.V0AbsoluteAddressing
	case first.Kind == OperandI && second.Kind == OperandAddress:
		return chip8.IAbsoluteAddressing
	case first.Kind == OperandRegister && second.Kind == OperandDT:
		return chip8.RegisterDTAddressing
	case first.Kind == OperandRegister && second.Kind == OperandK:
		return chip8.RegisterKAddressing
	case first.Kind == OperandDT && second.Kind == OperandRegister:
		return chip8.DTRegisterAddressing
	case first.Kind == OperandST && second.Kind == OperandRegister:
		return chip8.STRegisterAddressing
	case first.Kind == OperandF && second.Kind == OperandRegister:
		return chip8.FRegisterAddressing
	case first.Kind == OperandB && second.Kind == OperandRegister:
		return chip8.BRegisterAddressing
	case first.Kind == OperandI && second.Kind == OperandRegister:
		return chip8.IRegisterAddressing
	case first.Kind == OperandIndirectI && second.Kind == OperandRegister:
		return chip8.IIndirectRegisterAddressing
	case first.Kind == OperandRegister && second.Kind == OperandIndirectI:
		return chip8.RegisterIndirectIAddressing
	default:
		return chip8.NoAddressing
	}
}

func validateResolved(resolved ResolvedInstruction) error {
	addressing, err := resolveOperands(resolved.Instruction, resolved.Operands)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidInstruction, err)
	}
	if addressing != resolved.Addressing {
		return fmt.Errorf(
			"%w: operand mode %d does not match retained mode %d",
			ErrInvalidInstruction,
			addressing,
			resolved.Addressing,
		)
	}
	for index, operand := range resolved.Operands {
		if operand.Kind == OperandRegister && operand.Register > 0xf {
			return fmt.Errorf("%w: register %d exceeds VF", ErrInvalidInstruction, operand.Register)
		}
		maximum, valueBearing := operandMaximum(operand.Kind)
		if !valueBearing {
			if operand.Value != nil {
				return fmt.Errorf("%w: operand %d unexpectedly has a value", ErrInvalidInstruction, index)
			}
			continue
		}
		if operand.Value == nil {
			return fmt.Errorf("%w: operand %d has no value", ErrInvalidInstruction, index)
		}
		if value, ok := ast.NumberValue(operand.Value); ok && value > maximum {
			return fmt.Errorf("%w: operand %d value %d exceeds 0x%X", ErrInvalidInstruction, index, value, maximum)
		}
	}
	return nil
}

func operandMaximum(kind OperandKind) (uint64, bool) {
	switch kind {
	case OperandByte:
		return math.MaxUint8, true
	case OperandAddress:
		return 0xfff, true
	case OperandNibble:
		return 0xf, true
	default:
		return 0, false
	}
}

func resolvedFromParsed(instruction ast.Instruction, details *chip8.Instruction) (ast.Instruction, error) {
	operands, err := operandsFromParsed(instruction)
	if err != nil {
		return ast.Instruction{}, err
	}
	resolved := ResolvedInstruction{
		Instruction: details,
		Addressing:  chip8.Mode(instruction.Addressing),
		Operands:    operands,
	}
	if err := validateResolved(resolved); err != nil {
		return ast.Instruction{}, err
	}
	instruction.Argument = ast.NewInstructionArgument(resolved)
	return instruction, nil
}

func operandsFromParsed(instruction ast.Instruction) (Operands, error) { //nolint:cyclop // conversion mirrors all parser modes
	addressing := chip8.Mode(instruction.Addressing)
	switch addressing {
	case chip8.ImpliedAddressing:
		return nil, nil
	case chip8.AbsoluteAddressing:
		return Operands{AddressOperand(instruction.Argument.Copy())}, nil
	case chip8.V0AbsoluteAddressing:
		return Operands{RegisterOperand(0), AddressOperand(instruction.Argument.Copy())}, nil
	case chip8.IAbsoluteAddressing:
		return Operands{SpecialOperand(OperandI), AddressOperand(instruction.Argument.Copy())}, nil
	case chip8.RegisterAddressing:
		return singleParsedRegister(instruction.Argument)
	case chip8.RegisterValueAddressing:
		return parsedRegisterValue(instruction)
	case chip8.RegisterRegisterAddressing:
		value, ok := ast.NumberValue(instruction.Argument)
		if !ok {
			return nil, fmt.Errorf("%w: register pair is not numeric", ErrInvalidInstruction)
		}
		return Operands{RegisterOperand(byte(value >> 4)), RegisterOperand(byte(value & 0xf))}, nil
	case chip8.RegisterRegisterNibbleAddressing:
		argument, ok := instruction.Argument.(ast.RegisterRegisterValue)
		if !ok {
			return nil, fmt.Errorf("%w: unexpected draw argument %T", ErrInvalidInstruction, instruction.Argument)
		}
		return Operands{
			RegisterOperand(argument.Register1),
			RegisterOperand(argument.Register2),
			NibbleOperand(argument.Value.Copy()),
		}, nil
	default:
		return parsedSpecialOperands(addressing, instruction.Argument)
	}
}

func singleParsedRegister(argument ast.Node) (Operands, error) {
	value, ok := ast.NumberValue(argument)
	if !ok {
		return nil, fmt.Errorf("%w: register is not numeric", ErrInvalidInstruction)
	}
	return Operands{RegisterOperand(byte(value))}, nil
}

func parsedRegisterValue(instruction ast.Instruction) (Operands, error) {
	if instruction.Name == chip8.SkpName || instruction.Name == chip8.SknpName {
		value, ok := ast.NumberValue(instruction.Argument)
		if !ok {
			return nil, fmt.Errorf("%w: key register is not numeric", ErrInvalidInstruction)
		}
		return Operands{RegisterOperand(byte(value >> 8))}, nil
	}
	argument, ok := instruction.Argument.(ast.RegisterValue)
	if !ok {
		return nil, fmt.Errorf("%w: unexpected register-value argument %T", ErrInvalidInstruction, instruction.Argument)
	}
	return Operands{RegisterOperand(argument.Register), ByteOperand(argument.Value.Copy())}, nil
}

func parsedSpecialOperands(addressing chip8.Mode, argument ast.Node) (Operands, error) {
	registers, err := singleParsedRegister(argument)
	if err != nil {
		return nil, err
	}
	register := registers[0]
	switch addressing {
	case chip8.RegisterDTAddressing:
		return Operands{register, SpecialOperand(OperandDT)}, nil
	case chip8.RegisterKAddressing:
		return Operands{register, SpecialOperand(OperandK)}, nil
	case chip8.DTRegisterAddressing:
		return Operands{SpecialOperand(OperandDT), register}, nil
	case chip8.STRegisterAddressing:
		return Operands{SpecialOperand(OperandST), register}, nil
	case chip8.FRegisterAddressing:
		return Operands{SpecialOperand(OperandF), register}, nil
	case chip8.BRegisterAddressing:
		return Operands{SpecialOperand(OperandB), register}, nil
	case chip8.IRegisterAddressing:
		return Operands{SpecialOperand(OperandI), register}, nil
	case chip8.IIndirectRegisterAddressing:
		return Operands{SpecialOperand(OperandIndirectI), register}, nil
	case chip8.RegisterIndirectIAddressing:
		return Operands{register, SpecialOperand(OperandIndirectI)}, nil
	default:
		return nil, fmt.Errorf("%w: mode %d", ErrUnsupportedAddressing, addressing)
	}
}
