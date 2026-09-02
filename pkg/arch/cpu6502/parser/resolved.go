package parser

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/retroenv/retroasm/pkg/number"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu6502"
)

var (
	// ErrInvalidInstruction indicates inconsistent typed CPU6502 instruction state.
	ErrInvalidInstruction = errors.New("invalid typed CPU6502 instruction")
	// ErrUnsupportedAddressing indicates an operand shape unavailable for a mnemonic.
	ErrUnsupportedAddressing = errors.New("unsupported CPU6502 addressing")
)

// ResolvedInstruction is the typed projection of a compatibility CPU6502 AST instruction.
// The AST keeps its existing argument/modifier layout until legacy optimizer consumers migrate.
type ResolvedInstruction struct {
	Instruction *cpu6502.Instruction
	Addressing  cpu6502.AddressingMode
	Operands    Operands
}

// ResolveInstruction returns the typed projection of a CPU6502 AST instruction.
func ResolveInstruction(instruction ast.Instruction, expected *cpu6502.Instruction) (ResolvedInstruction, error) {
	if expected == nil || !strings.EqualFold(strings.TrimSpace(instruction.Name), expected.Name) {
		return ResolvedInstruction{}, fmt.Errorf("%w: mnemonic %q", ErrInvalidInstruction, instruction.Name)
	}
	operands, err := operandsFromInstruction(instruction)
	if err != nil {
		return ResolvedInstruction{}, err
	}
	resolved := ResolvedInstruction{
		Instruction: expected,
		Addressing:  cpu6502.AddressingMode(instruction.Addressing),
		Operands:    operands,
	}
	if err := validateResolved(resolved); err != nil {
		return ResolvedInstruction{}, err
	}
	return resolved, nil
}

func operandsFromInstruction(instruction ast.Instruction) (Operands, error) {
	addressing := cpu6502.AddressingMode(instruction.Addressing)
	if addressing == cpu6502.ImpliedAddressing {
		return nil, nil
	}
	if addressing == cpu6502.AccumulatorAddressing {
		return Operands{AccumulatorOperand()}, nil
	}
	if instruction.Argument == nil {
		return nil, fmt.Errorf("%w: missing operand", ErrInvalidInstruction)
	}

	kind, size, err := operandShape(addressing)
	if err != nil {
		return nil, err
	}
	return Operands{{
		Kind:      kind,
		Size:      size,
		Value:     instruction.Argument.Copy(),
		Modifiers: slices.Clone(instruction.Modifier),
	}}, nil
}

func operandShape(addressing cpu6502.AddressingMode) (OperandKind, AddressSize, error) {
	shape, ok := addressingOperandShapes[addressing]
	if !ok {
		return OperandInvalid, AddressDefault, fmt.Errorf("%w: mode %d", ErrUnsupportedAddressing, addressing)
	}
	return shape.kind, shape.size, nil
}

type operandShapeInfo struct {
	kind OperandKind
	size AddressSize
}

var addressingOperandShapes = map[cpu6502.AddressingMode]operandShapeInfo{
	cpu6502.ImmediateAddressing: {OperandImmediate, AddressDefault},
	cpu6502.RelativeAddressing:  {OperandAddress, AddressDefault},
	cpu6502.ZeroPageAddressing:  {OperandAddress, AddressZeroPage},
	cpu6502.AbsoluteAddressing:  {OperandAddress, AddressAbsolute},
	AbsoluteZeroPageAddressing:  {OperandAddress, AddressDefault},
	cpu6502.ZeroPageXAddressing: {OperandIndexedX, AddressZeroPage},
	cpu6502.AbsoluteXAddressing: {OperandIndexedX, AddressAbsolute},
	XAddressing:                 {OperandIndexedX, AddressDefault},
	cpu6502.ZeroPageYAddressing: {OperandIndexedY, AddressZeroPage},
	cpu6502.AbsoluteYAddressing: {OperandIndexedY, AddressAbsolute},
	YAddressing:                 {OperandIndexedY, AddressDefault},
	cpu6502.IndirectAddressing:  {OperandIndirect, AddressAbsolute},
	cpu6502.IndirectXAddressing: {OperandIndexedXIndirect, AddressZeroPage},
	cpu6502.IndirectYAddressing: {OperandIndirectIndexedY, AddressZeroPage},
}

func resolveOperands(instruction *cpu6502.Instruction, operands Operands) (cpu6502.AddressingMode, error) {
	if instruction == nil {
		return cpu6502.NoAddressing, ErrUnsupportedAddressing
	}
	switch len(operands) {
	case 0:
		if instruction.HasAddressing(cpu6502.ImpliedAddressing) {
			return cpu6502.ImpliedAddressing, nil
		}
	case 1:
		return resolveOperand(instruction, operands[0])
	}
	return cpu6502.NoAddressing, ErrUnsupportedAddressing
}

func resolveOperand(instruction *cpu6502.Instruction, operand Operand) (cpu6502.AddressingMode, error) {
	var addressing cpu6502.AddressingMode
	switch operand.Kind {
	case OperandAccumulator:
		addressing = cpu6502.AccumulatorAddressing
	case OperandImmediate:
		addressing = cpu6502.ImmediateAddressing
	case OperandAddress:
		addressing = resolveAddress(instruction, operand)
	case OperandIndexedX:
		addressing = resolveIndexed(instruction, operand, cpu6502.ZeroPageXAddressing, cpu6502.AbsoluteXAddressing, XAddressing)
	case OperandIndexedY:
		addressing = resolveIndexed(instruction, operand, cpu6502.ZeroPageYAddressing, cpu6502.AbsoluteYAddressing, YAddressing)
	case OperandIndirect:
		addressing = cpu6502.IndirectAddressing
	case OperandIndexedXIndirect:
		addressing = cpu6502.IndirectXAddressing
	case OperandIndirectIndexedY:
		addressing = cpu6502.IndirectYAddressing
	default:
		return cpu6502.NoAddressing, ErrUnsupportedAddressing
	}
	if !supportsAddressing(instruction, addressing) {
		return cpu6502.NoAddressing, ErrUnsupportedAddressing
	}
	return addressing, nil
}

func resolveAddress(instruction *cpu6502.Instruction, operand Operand) cpu6502.AddressingMode {
	if instruction.HasAddressing(cpu6502.RelativeAddressing) {
		return cpu6502.RelativeAddressing
	}
	if operand.Size == AddressZeroPage {
		return cpu6502.ZeroPageAddressing
	}
	if operand.Size == AddressAbsolute {
		return cpu6502.AbsoluteAddressing
	}
	return resolveMemorySize(
		instruction,
		operand.Value,
		cpu6502.ZeroPageAddressing,
		cpu6502.AbsoluteAddressing,
		AbsoluteZeroPageAddressing,
	)
}

func resolveIndexed(
	instruction *cpu6502.Instruction,
	operand Operand,
	zeroPage, absolute, ambiguous cpu6502.AddressingMode,
) cpu6502.AddressingMode {

	if operand.Size == AddressZeroPage {
		return zeroPage
	}
	if operand.Size == AddressAbsolute {
		return absolute
	}
	return resolveMemorySize(instruction, operand.Value, zeroPage, absolute, ambiguous)
}

func resolveMemorySize(
	instruction *cpu6502.Instruction,
	value ast.Node,
	zeroPage, absolute, ambiguous cpu6502.AddressingMode,
) cpu6502.AddressingMode {

	hasZeroPage := instruction.HasAddressing(zeroPage)
	hasAbsolute := instruction.HasAddressing(absolute)
	if numberValue, ok := ast.NumberValue(value); ok {
		if numberValue <= math.MaxUint8 && hasZeroPage {
			return zeroPage
		}
		if hasAbsolute {
			return absolute
		}
		if hasZeroPage {
			return zeroPage
		}
	}
	switch {
	case hasZeroPage && hasAbsolute:
		return ambiguous
	case hasAbsolute:
		return absolute
	case hasZeroPage:
		return zeroPage
	default:
		return cpu6502.NoAddressing
	}
}

func supportsAddressing(instruction *cpu6502.Instruction, addressing cpu6502.AddressingMode) bool {
	switch addressing {
	case AbsoluteZeroPageAddressing:
		return instruction.HasAddressing(cpu6502.AbsoluteAddressing) && instruction.HasAddressing(cpu6502.ZeroPageAddressing)
	case XAddressing:
		return instruction.HasAddressing(cpu6502.AbsoluteXAddressing) && instruction.HasAddressing(cpu6502.ZeroPageXAddressing)
	case YAddressing:
		return instruction.HasAddressing(cpu6502.AbsoluteYAddressing) && instruction.HasAddressing(cpu6502.ZeroPageYAddressing)
	default:
		return instruction.HasAddressing(addressing)
	}
}

func validateResolved(resolved ResolvedInstruction) error {
	addressing, err := resolveOperands(resolved.Instruction, resolved.Operands)
	if err != nil {
		return fmt.Errorf("%w: resolving operands: %w", ErrInvalidInstruction, err)
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
		if operand.Kind != OperandAccumulator && operand.Value == nil {
			return fmt.Errorf("%w: operand %d has no value", ErrInvalidInstruction, index)
		}
		for _, modifier := range operand.Modifiers {
			if modifier.Operator.Operator != "+" && modifier.Operator.Operator != "-" {
				return fmt.Errorf("%w: unsupported modifier operator %q", ErrInvalidInstruction, modifier.Operator.Operator)
			}
			if _, err := number.Parse(modifier.Value); err != nil {
				return fmt.Errorf("%w: modifier %q: %w", ErrInvalidInstruction, modifier.Value, err)
			}
		}
	}
	return validateWidths(resolved)
}

func validateWidths(resolved ResolvedInstruction) error {
	maximum := uint64(math.MaxUint16)
	switch resolved.Addressing {
	case cpu6502.ImmediateAddressing, cpu6502.ZeroPageAddressing,
		cpu6502.ZeroPageXAddressing, cpu6502.ZeroPageYAddressing,
		cpu6502.IndirectXAddressing, cpu6502.IndirectYAddressing:
		maximum = math.MaxUint8
	}
	for index, operand := range resolved.Operands {
		if value, ok := ast.NumberValue(operand.Value); ok && value > maximum {
			return fmt.Errorf("%w: operand %d value %d exceeds 0x%X", ErrInvalidInstruction, index, value, maximum)
		}
	}
	return nil
}
