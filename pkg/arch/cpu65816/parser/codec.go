package parser

import (
	"errors"
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/number"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu65816"
)

// BuildInstruction constructs a typed CPU65816 instruction in the default 8-bit state.
func BuildInstruction(
	mnemonic string,
	instruction *cpu65816.Instruction,
	operands Operands,
) (ast.Instruction, error) {

	built, _, err := BuildInstructionWithState(mnemonic, instruction, operands, DefaultState())
	return built, err
}

// BuildInstructionWithState constructs a typed CPU65816 instruction and returns its successor state.
func BuildInstructionWithState(
	mnemonic string,
	instruction *cpu65816.Instruction,
	operands Operands,
	state State,
) (ast.Instruction, State, error) {

	var zero State
	mnemonic = strings.ToLower(strings.TrimSpace(mnemonic))
	if instruction == nil || mnemonic == "" || instruction.Name != mnemonic {
		return ast.Instruction{}, zero, fmt.Errorf("%w: mnemonic %q", ErrInvalidInstruction, mnemonic)
	}
	if err := state.validate(); err != nil {
		return ast.Instruction{}, zero, fmt.Errorf("%w: %w", ErrInvalidInstruction, err)
	}
	operands = normalizeOperands(instruction, copyOperands(operands))
	addressing, err := resolveOperands(instruction, operands)
	if err != nil {
		return ast.Instruction{}, zero, fmt.Errorf("%w: resolving %s operands: %w", ErrInvalidInstruction, mnemonic, err)
	}
	resolved := ResolvedInstruction{
		Instruction: instruction,
		Addressing:  addressing,
		Operands:    operands,
		State:       state,
	}
	if err := validateResolved(resolved); err != nil {
		return ast.Instruction{}, zero, err
	}

	argument := ast.NewInstructionArgument(resolved)
	built := ast.NewInstruction(mnemonic, int(addressing), argument, nil)
	return built, nextState(state, instruction, operands), nil
}

// ValidateInstruction checks typed metadata against a mnemonic's retrogolib definition.
func ValidateInstruction(instruction ast.Instruction, expected *cpu65816.Instruction) error {
	resolved, err := resolvedArgument(instruction.Argument)
	if err != nil {
		return err
	}
	if expected == nil || resolved.Instruction != expected {
		return fmt.Errorf("%w: retained instruction does not match mnemonic", ErrInvalidInstruction)
	}
	if !strings.EqualFold(strings.TrimSpace(instruction.Name), expected.Name) {
		return fmt.Errorf(
			"%w: mnemonic %q does not match %q",
			ErrInvalidInstruction,
			instruction.Name,
			expected.Name,
		)
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
		return fmt.Errorf("%w: modifiers must be retained on typed operands", ErrInvalidInstruction)
	}
	return validateResolved(resolved)
}

// FormatOptions controls deterministic CPU65816 instruction spelling.
type FormatOptions struct {
	Indent           string
	Uppercase        bool
	ByteHexDigits    int
	LongHexDigits    int
	MinimumHexDigits int
	WordHexDigits    int
}

// FormatInstruction returns one deterministic, parseable CPU65816 instruction line.
func FormatInstruction(instruction ast.Instruction) (string, error) {
	return FormatInstructionWithOptions(instruction, FormatOptions{})
}

// FormatInstructionWithOptions formats one typed CPU65816 instruction.
func FormatInstructionWithOptions(instruction ast.Instruction, options FormatOptions) (string, error) {
	resolved, err := resolvedArgument(instruction.Argument)
	if err != nil {
		return "", err
	}
	mnemonic := strings.ToLower(strings.TrimSpace(instruction.Name))
	if mnemonic == "" {
		return "", fmt.Errorf("%w: missing mnemonic", ErrInvalidInstruction)
	}
	if options.Uppercase {
		mnemonic = strings.ToUpper(mnemonic)
	}
	if len(resolved.Operands) == 0 {
		return options.Indent + mnemonic, nil
	}

	formatted := make([]string, len(resolved.Operands))
	for index, operand := range resolved.Operands {
		formatted[index], err = formatOperand(resolved, operand, options)
		if err != nil {
			return "", fmt.Errorf("formatting operand %d: %w", index, err)
		}
	}
	return options.Indent + mnemonic + " " + strings.Join(formatted, ","), nil
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

func normalizeOperands(instruction *cpu65816.Instruction, operands Operands) Operands {
	if len(operands) == 0 && !instruction.HasAddressing(cpu65816.ImpliedAddressing) &&
		instruction.HasAddressing(cpu65816.AccumulatorAddressing) {

		return Operands{AccumulatorOperand()}
	}
	return operands
}

func formatOperand(resolved ResolvedInstruction, operand Operand, options FormatOptions) (string, error) {
	if operand.Kind == OperandAccumulator {
		return formatRegister("a", options), nil
	}
	if operand.Value == nil {
		return "", errors.New("operand value is missing")
	}

	minimumDigits := operandHexDigits(resolved, operand, options)
	value, err := formatValue(operand.Value, minimumDigits)
	if err != nil {
		return "", err //nolint:wrapcheck // AST formatter already identifies the operand value
	}
	value, err = appendModifiers(value, operand.Modifiers)
	if err != nil {
		return "", err
	}
	prefix := addressPrefix(operand.Size, options)
	registerX := formatRegister("x", options)
	registerY := formatRegister("y", options)
	registerS := formatRegister("s", options)

	switch operand.Kind {
	case OperandImmediate:
		return "#" + value, nil
	case OperandAddress, OperandBlockMoveBank:
		return prefix + value, nil
	case OperandIndexedX:
		return prefix + value + "," + registerX, nil
	case OperandIndexedY:
		return prefix + value + "," + registerY, nil
	case OperandIndirect:
		return prefix + "(" + value + ")", nil
	case OperandIndexedXIndirect:
		return prefix + "(" + value + "," + registerX + ")", nil
	case OperandIndirectIndexedY:
		return prefix + "(" + value + ")," + registerY, nil
	case OperandIndirectLong:
		return prefix + "[" + value + "]", nil
	case OperandIndirectLongIndexedY:
		return prefix + "[" + value + "]," + registerY, nil
	case OperandStackRelative:
		return value + "," + registerS, nil
	case OperandStackRelativeIndirectIndexedY:
		return "(" + value + "," + registerS + ")," + registerY, nil
	default:
		return "", fmt.Errorf("%w: operand kind %d", ErrInvalidInstruction, operand.Kind)
	}
}

func formatValue(value ast.Node, minimumHexDigits int) (string, error) {
	if numberValue, ok := ast.NumberValue(value); ok {
		if minimumHexDigits > 0 {
			return fmt.Sprintf("$%0*X", minimumHexDigits, numberValue), nil
		}
		return fmt.Sprintf("$%X", numberValue), nil
	}
	formatted, err := ast.FormatValue(value, ast.ValueFormatOptions{MinimumHexDigits: minimumHexDigits})
	if err != nil {
		return "", fmt.Errorf("formatting CPU65816 value: %w", err)
	}
	return formatted, nil
}

func operandHexDigits(resolved ResolvedInstruction, operand Operand, options FormatOptions) int {
	if options.MinimumHexDigits > 0 {
		return options.MinimumHexDigits
	}
	byteDigits := options.ByteHexDigits
	if byteDigits == 0 {
		byteDigits = 2
	}
	wordDigits := options.WordHexDigits
	if wordDigits == 0 {
		wordDigits = 4
	}
	longDigits := options.LongHexDigits
	if longDigits == 0 {
		longDigits = 6
	}

	if operand.Kind == OperandImmediate {
		if immediateWidth(resolved.Instruction, resolved.State) == WidthWord {
			return wordDigits
		}
		return byteDigits
	}
	switch operand.Size {
	case AddressDirectPage:
		return byteDigits
	case AddressAbsolute:
		return wordDigits
	case AddressLong:
		return longDigits
	default:
		if operand.Kind == OperandBlockMoveBank ||
			resolved.Addressing == cpu65816.StackRelativeAddressing ||
			resolved.Addressing == cpu65816.StackRelativeIndirectIndexedYAddressing {

			return byteDigits
		}
		return options.MinimumHexDigits
	}
}

func appendModifiers(value string, modifiers []ast.Modifier) (string, error) {
	var builder strings.Builder
	builder.WriteString(value)
	for _, modifier := range modifiers {
		parsed, err := number.Parse(modifier.Value)
		if err != nil {
			return "", fmt.Errorf("parsing modifier %q: %w", modifier.Value, err)
		}
		builder.WriteString(modifier.Operator.Operator)
		formatted, err := formatValue(ast.NewNumber(parsed), 0)
		if err != nil {
			return "", err //nolint:wrapcheck // numeric AST values always use the shared formatter
		}
		builder.WriteString(formatted)
	}
	return builder.String(), nil
}

func addressPrefix(size AddressSize, options FormatOptions) string {
	var prefix string
	switch size {
	case AddressDirectPage:
		prefix = "z:"
	case AddressAbsolute:
		prefix = "a:"
	case AddressLong:
		prefix = "f:"
	}
	if options.Uppercase {
		return strings.ToUpper(prefix)
	}
	return prefix
}

func formatRegister(register string, options FormatOptions) string {
	if options.Uppercase {
		return strings.ToUpper(register)
	}
	return register
}
