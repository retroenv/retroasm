package parser

import (
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/chip8"
)

// FormatOptions controls deterministic CHIP-8 instruction spelling.
type FormatOptions struct {
	Indent            string
	Uppercase         bool
	UppercaseMnemonic bool
	UppercaseOperands bool
	SpaceAfterComma   bool
}

// BuildInstruction constructs a typed CHIP-8 instruction without parsing text.
func BuildInstruction(
	mnemonic string,
	instruction *chip8.Instruction,
	operands Operands,
) (ast.Instruction, error) {

	mnemonic = strings.ToLower(strings.TrimSpace(mnemonic))
	if instruction == nil || mnemonic == "" || instruction.Name != mnemonic {
		return ast.Instruction{}, fmt.Errorf("%w: mnemonic %q", ErrInvalidInstruction, mnemonic)
	}
	operands = copyOperands(operands)
	addressing, err := resolveOperands(instruction, operands)
	if err != nil {
		return ast.Instruction{}, fmt.Errorf("%w: resolving %s operands: %w", ErrInvalidInstruction, mnemonic, err)
	}
	resolved := ResolvedInstruction{
		Instruction: instruction,
		Addressing:  addressing,
		Operands:    operands,
	}
	if err := validateResolved(resolved); err != nil {
		return ast.Instruction{}, err
	}
	argument := ast.NewInstructionArgument(resolved)
	return ast.NewInstruction(mnemonic, int(addressing), argument, nil), nil
}

// ValidateInstruction checks typed metadata against a mnemonic's retrogolib definition.
func ValidateInstruction(instruction ast.Instruction, expected *chip8.Instruction) error {
	resolved, err := resolvedArgument(instruction.Argument)
	if err != nil {
		return err
	}
	if expected == nil || resolved.Instruction != expected {
		return fmt.Errorf("%w: retained instruction does not match mnemonic", ErrInvalidInstruction)
	}
	if !strings.EqualFold(strings.TrimSpace(instruction.Name), expected.Name) {
		return fmt.Errorf("%w: mnemonic %q does not match %q", ErrInvalidInstruction, instruction.Name, expected.Name)
	}
	if instruction.Addressing != int(resolved.Addressing) {
		return fmt.Errorf(
			"%w: AST addressing %d does not match retained addressing %d",
			ErrInvalidInstruction,
			instruction.Addressing,
			resolved.Addressing,
		)
	}
	if len(instruction.Modifier) != 0 {
		return fmt.Errorf("%w: instruction modifiers are not supported", ErrInvalidInstruction)
	}
	return validateResolved(resolved)
}

// FormatInstruction returns one deterministic, parseable CHIP-8 instruction line.
func FormatInstruction(instruction ast.Instruction) (string, error) {
	return FormatInstructionWithOptions(instruction, FormatOptions{})
}

// FormatInstructionWithOptions formats one typed CHIP-8 instruction.
func FormatInstructionWithOptions(instruction ast.Instruction, options FormatOptions) (string, error) {
	resolved, err := resolvedArgument(instruction.Argument)
	if err != nil {
		return "", err
	}
	mnemonic := strings.ToLower(strings.TrimSpace(instruction.Name))
	if options.Uppercase || options.UppercaseMnemonic {
		mnemonic = strings.ToUpper(mnemonic)
	}
	formatted := make([]string, len(resolved.Operands))
	for index, operand := range resolved.Operands {
		formatted[index], err = formatOperand(operand, options)
		if err != nil {
			return "", fmt.Errorf("formatting operand %d: %w", index, err)
		}
	}
	if len(formatted) == 0 {
		return options.Indent + mnemonic, nil
	}
	separator := ","
	if options.SpaceAfterComma {
		separator = ", "
	}
	return options.Indent + mnemonic + " " + strings.Join(formatted, separator), nil
}

func resolvedArgument(argument ast.Node) (ResolvedInstruction, error) {
	var value any
	switch typed := argument.(type) {
	case ast.InstructionArgument:
		value = typed.Value
	case *ast.InstructionArgument:
		if typed == nil {
			return ResolvedInstruction{}, fmt.Errorf("%w: nil instruction argument", ErrInvalidInstruction)
		}
		value = typed.Value
	default:
		return ResolvedInstruction{}, fmt.Errorf("%w: unexpected argument %T", ErrInvalidInstruction, argument)
	}
	switch resolved := value.(type) {
	case ResolvedInstruction:
		return resolved, nil
	case *ResolvedInstruction:
		if resolved != nil {
			return *resolved, nil
		}
	}
	return ResolvedInstruction{}, fmt.Errorf("%w: unexpected resolved argument %T", ErrInvalidInstruction, value)
}

func formatOperand(operand Operand, options FormatOptions) (string, error) {
	keyword := func(value string) string {
		if options.Uppercase || options.UppercaseOperands {
			return strings.ToUpper(value)
		}
		return value
	}
	switch operand.Kind {
	case OperandRegister:
		return formatRegister(operand.Register, options), nil
	case OperandByte:
		return formatValue(operand.Value, 2)
	case OperandAddress:
		return formatValue(operand.Value, 3)
	case OperandNibble:
		return formatValue(operand.Value, 1)
	case OperandI:
		return keyword("i"), nil
	case OperandDT:
		return keyword("dt"), nil
	case OperandST:
		return keyword("st"), nil
	case OperandF:
		return keyword("f"), nil
	case OperandB:
		return keyword("b"), nil
	case OperandK:
		return keyword("k"), nil
	case OperandIndirectI:
		return "[" + keyword("i") + "]", nil
	default:
		return "", fmt.Errorf("%w: operand kind %d", ErrInvalidInstruction, operand.Kind)
	}
}

func formatRegister(register byte, options FormatOptions) string {
	formatted := fmt.Sprintf("v%x", register)
	if options.Uppercase || options.UppercaseOperands {
		return strings.ToUpper(formatted)
	}
	return formatted
}

func formatValue(value ast.Node, minimumDigits int) (string, error) {
	formatted, err := ast.FormatValue(value, ast.ValueFormatOptions{MinimumHexDigits: minimumDigits})
	if err != nil {
		return "", fmt.Errorf("formatting CHIP-8 value: %w", err)
	}
	return formatted, nil
}
