package parser

import (
	"fmt"
	"slices"
	"strings"

	"github.com/retroenv/retroasm/pkg/number"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu6502"
)

// BuildInstruction constructs a typed CPU6502 instruction without parsing text.
// It preserves the compatibility AST argument layout used by existing CPU6502 passes.
func BuildInstruction(
	mnemonic string,
	instruction *cpu6502.Instruction,
	operands Operands,
) (ast.Instruction, error) {

	mnemonic = strings.ToLower(strings.TrimSpace(mnemonic))
	if instruction == nil || mnemonic == "" || instruction.Name != mnemonic {
		return ast.Instruction{}, fmt.Errorf("%w: mnemonic %q", ErrInvalidInstruction, mnemonic)
	}
	operands = normalizeOperands(instruction, copyOperands(operands))
	addressing, err := resolveOperands(instruction, operands)
	if err != nil {
		return ast.Instruction{}, fmt.Errorf("%w: resolving %s operands: %w", ErrInvalidInstruction, mnemonic, err)
	}
	resolved := ResolvedInstruction{Instruction: instruction, Addressing: addressing, Operands: operands}
	if err := validateResolved(resolved); err != nil {
		return ast.Instruction{}, err
	}

	var argument ast.Node
	var modifiers []ast.Modifier
	if len(operands) == 2 {
		argument = ast.NewInstructionArguments(operands[0].Value.Copy(), operands[1].Value.Copy())
	} else if len(operands) == 1 && operands[0].Kind != OperandAccumulator {
		argument = operands[0].Value.Copy()
		modifiers = slices.Clone(operands[0].Modifiers)
	}
	return ast.NewInstruction(mnemonic, int(addressing), argument, modifiers), nil
}

// ValidateInstruction checks a compatibility AST instruction through its typed projection.
func ValidateInstruction(instruction ast.Instruction, expected *cpu6502.Instruction) error {
	_, err := ResolveInstruction(instruction, expected)
	return err
}

// FormatOptions controls deterministic CPU6502 instruction spelling.
type FormatOptions struct {
	Indent           string
	Uppercase        bool
	ByteHexDigits    int
	MinimumHexDigits int
	WordHexDigits    int
}

// FormatInstruction returns one deterministic, parseable CPU6502 instruction line.
func FormatInstruction(instruction ast.Instruction, expected *cpu6502.Instruction) (string, error) {
	return FormatInstructionWithOptions(instruction, expected, FormatOptions{})
}

// FormatInstructionWithOptions formats one typed CPU6502 projection.
func FormatInstructionWithOptions(
	instruction ast.Instruction,
	expected *cpu6502.Instruction,
	options FormatOptions,
) (string, error) {

	resolved, err := ResolveInstruction(instruction, expected)
	if err != nil {
		return "", err
	}
	mnemonic := strings.ToLower(strings.TrimSpace(instruction.Name))
	if options.Uppercase {
		mnemonic = strings.ToUpper(mnemonic)
	}
	if len(resolved.Operands) == 0 {
		return options.Indent + mnemonic, nil
	}
	formatted := make([]string, 0, len(resolved.Operands))
	for _, operand := range resolved.Operands {
		value, err := formatOperand(resolved, operand, options)
		if err != nil {
			return "", err
		}
		formatted = append(formatted, value)
	}
	return options.Indent + mnemonic + " " + strings.Join(formatted, ","), nil
}

func normalizeOperands(instruction *cpu6502.Instruction, operands Operands) Operands {
	if len(operands) == 0 && !instruction.HasAddressing(cpu6502.ImpliedAddressing) &&
		instruction.HasAddressing(cpu6502.AccumulatorAddressing) {

		return Operands{AccumulatorOperand()}
	}
	return operands
}

func formatOperand(resolved ResolvedInstruction, operand Operand, options FormatOptions) (string, error) {
	if operand.Kind == OperandAccumulator {
		return formatRegister("a", options), nil
	}
	minimumDigits := operandHexDigits(resolved, operand, options)
	value, err := formatValue(operand.Value, minimumDigits)
	if err != nil {
		return "", err
	}
	value, err = appendModifiers(value, operand.Modifiers)
	if err != nil {
		return "", err
	}
	prefix := addressPrefix(operand.Size, options)
	registerX := formatRegister("x", options)
	registerY := formatRegister("y", options)

	switch operand.Kind {
	case OperandImmediate:
		return "#" + value, nil
	case OperandAddress:
		return prefix + value, nil
	case OperandRelativeTarget:
		return value, nil
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
	default:
		return "", fmt.Errorf("%w: operand kind %d", ErrInvalidInstruction, operand.Kind)
	}
}

func formatValue(value ast.Node, minimumDigits int) (string, error) {
	if numberValue, ok := ast.NumberValue(value); ok {
		if minimumDigits > 0 {
			return fmt.Sprintf("$%0*X", minimumDigits, numberValue), nil
		}
		return fmt.Sprintf("$%X", numberValue), nil
	}
	formatted, err := ast.FormatValue(value, ast.ValueFormatOptions{MinimumHexDigits: minimumDigits})
	if err != nil {
		return "", fmt.Errorf("formatting CPU6502 value: %w", err)
	}
	return formatted, nil
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
			return "", err
		}
		builder.WriteString(formatted)
	}
	return builder.String(), nil
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
	if operand.Kind == OperandImmediate || operand.Size == AddressZeroPage ||
		resolved.Addressing == cpu6502.IndirectXAddressing ||
		resolved.Addressing == cpu6502.IndirectYAddressing ||
		resolved.Addressing == cpu6502.ZeroPageIndirectAddressing {

		return byteDigits
	}
	if operand.Size == AddressAbsolute {
		return wordDigits
	}
	return options.MinimumHexDigits
}

func addressPrefix(size AddressSize, options FormatOptions) string {
	var prefix string
	switch size {
	case AddressZeroPage:
		prefix = "z:"
	case AddressAbsolute:
		prefix = "a:"
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
