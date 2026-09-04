package parser

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/sm83"
)

var (
	// ErrInvalidInstruction indicates inconsistent or incomplete typed SM83 instruction state.
	ErrInvalidInstruction = errors.New("invalid typed SM83 instruction")
	// ErrUnsupportedValue indicates a value node that the SM83 formatter cannot represent.
	ErrUnsupportedValue = errors.New("unsupported SM83 operand value")
)

// FormatOptions controls deterministic SM83 instruction spelling without
// changing the typed instruction or its encoding.
type FormatOptions struct {
	Indent                 string
	Uppercase              bool
	MinimumHexDigits       int
	PairImmediateHexDigits int
	DecimalBitIndexes      bool
	DecimalSignedOffsets   bool
}

// BuildInstruction constructs and validates a typed SM83 instruction without parsing text.
func BuildInstruction(
	mnemonic string,
	variants []*sm83.Instruction,
	operands ...Operand,
) (ast.Instruction, error) {

	mnemonic = strings.ToLower(strings.TrimSpace(mnemonic))
	if mnemonic == "" {
		return ast.Instruction{}, fmt.Errorf("%w: missing mnemonic", ErrInvalidInstruction)
	}

	raw, err := rawOperands(operands)
	if err != nil {
		return ast.Instruction{}, fmt.Errorf("%w: %w", ErrInvalidInstruction, err)
	}
	resolved, err := resolveInstruction(mnemonic, variants, raw)
	if err != nil {
		return ast.Instruction{}, fmt.Errorf("%w: resolving '%s': %w", ErrInvalidInstruction, mnemonic, err)
	}
	resolved.Operands = copyOperands(operands)
	if err := validateResolvedValues(*resolved); err != nil {
		return ast.Instruction{}, fmt.Errorf("%w: %w", ErrInvalidInstruction, err)
	}

	argument := ast.NewInstructionArgument(*resolved)
	return ast.NewInstruction(mnemonic, int(resolved.Addressing), argument, nil), nil
}

// ValidateInstruction checks typed SM83 metadata against its source-level operands.
func ValidateInstruction(instruction ast.Instruction, variants []*sm83.Instruction) error {
	resolved, err := resolvedArgument(instruction.Argument)
	if err != nil {
		return err
	}
	if err := validateResolvedMetadata(instruction, resolved, variants); err != nil {
		return err
	}

	rebuilt, err := BuildInstruction(instruction.Name, variants, resolved.Operands...)
	if err != nil {
		return fmt.Errorf("%w: rebuilding operands: %w", ErrInvalidInstruction, err)
	}
	rebuiltResolved, err := resolvedArgument(rebuilt.Argument)
	if err != nil {
		return err
	}
	if rebuiltResolved.Instruction != resolved.Instruction ||
		rebuiltResolved.Addressing != resolved.Addressing ||
		!slices.Equal(rebuiltResolved.RegisterParams, resolved.RegisterParams) {

		return fmt.Errorf("%w: resolved variant does not match operands", ErrInvalidInstruction)
	}
	if !sameValues(rebuiltResolved.OperandValues, resolved.OperandValues) ||
		!sameOperands(rebuiltResolved.Operands, resolved.Operands) {

		return fmt.Errorf("%w: resolved values do not match operands", ErrInvalidInstruction)
	}
	return nil
}

// FormatInstruction returns one deterministic, parseable SM83 instruction line.
func FormatInstruction(instruction ast.Instruction) (string, error) {
	return FormatInstructionWithOptions(instruction, FormatOptions{})
}

// FormatInstructionWithOptions returns one deterministic, parseable SM83
// instruction line using the requested presentation policy.
func FormatInstructionWithOptions(
	instruction ast.Instruction,
	options FormatOptions,
) (string, error) {

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
		operandOptions := options
		decimalValue := options.DecimalBitIndexes && index == 0 &&
			(strings.EqualFold(mnemonic, sm83.BitName) ||
				strings.EqualFold(mnemonic, sm83.SetName) ||
				strings.EqualFold(mnemonic, sm83.ResName))
		if index == 1 && strings.EqualFold(mnemonic, sm83.LdName) &&
			len(resolved.Operands) == 2 && sm83PairRegisterOperand(resolved.Operands[0]) {

			operandOptions.MinimumHexDigits = options.PairImmediateHexDigits
		}
		signedValue := index == 1 && len(resolved.Operands) == 2 &&
			resolved.Instruction == sm83.AddSPE
		formatted[index], err = formatOperandWithOptions(operand, operandOptions, decimalValue, signedValue)
		if err != nil {
			return "", fmt.Errorf("formatting operand %d: %w", index, err)
		}
	}
	return options.Indent + mnemonic + " " + strings.Join(formatted, ","), nil
}

// FormatOperand returns the canonical SM83 spelling for one typed operand.
func FormatOperand(operand Operand) (string, error) {
	return formatOperandWithOptions(operand, FormatOptions{}, false, false)
}

func validateResolvedMetadata(
	instruction ast.Instruction,
	resolved ResolvedInstruction,
	variants []*sm83.Instruction,
) error {

	if resolved.Instruction == nil {
		return fmt.Errorf("%w: missing instruction variant", ErrInvalidInstruction)
	}
	if !slices.Contains(variants, resolved.Instruction) {
		return fmt.Errorf("%w: variant does not belong to mnemonic %q", ErrInvalidInstruction, instruction.Name)
	}
	if !strings.EqualFold(strings.TrimSpace(instruction.Name), resolved.Instruction.Name) {
		return fmt.Errorf(
			"%w: mnemonic %q does not match variant %q",
			ErrInvalidInstruction,
			instruction.Name,
			resolved.Instruction.Name,
		)
	}
	if instruction.Addressing != int(resolved.Addressing) {
		return fmt.Errorf(
			"%w: AST addressing %d does not match resolved addressing %d",
			ErrInvalidInstruction,
			instruction.Addressing,
			resolved.Addressing,
		)
	}
	if err := validateResolvedValues(resolved); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidInstruction, err)
	}
	return nil
}

func validateResolvedValues(resolved ResolvedInstruction) error {
	opcodeInfo, addressing, err := resolved.OpcodeInfo()
	if err != nil {
		return err
	}

	if isCBBitInstruction(resolved.Instruction) && len(resolved.OperandValues) > 0 {
		if value, ok := ast.NumberValue(resolved.OperandValues[0]); ok && value > 7 {
			return fmt.Errorf("bit number %d exceeds 7", value)
		}
	}
	if resolved.Instruction == sm83.LdHLSPOffset || resolved.Instruction == sm83.AddSPE {
		if len(resolved.OperandValues) != 1 {
			return errors.New("signed SP offset is missing")
		}
		value, ok := ast.NumberValue(resolved.OperandValues[0])
		if !ok {
			return errors.New("signed SP offset must be numeric")
		}
		if value > 0xff {
			return fmt.Errorf("signed SP offset %d exceeds byte", value)
		}
	}

	baseWidth := 1
	if opcodeInfo.Prefix != 0 {
		baseWidth++
	}
	remaining := int(opcodeInfo.Size) - baseWidth
	switch addressing {
	case sm83.ImmediateAddressing:
		if remaining == 1 {
			return validateNumberWidths(resolved.OperandValues, 0xff)
		}
		if remaining == 2 {
			return validateNumberWidths(resolved.OperandValues, 0xffff)
		}
	case sm83.ExtendedAddressing, sm83.RelativeAddressing:
		return validateNumberWidths(resolved.OperandValues, 0xffff)
	case sm83.RegisterIndirectAddressing:
		if remaining > 0 {
			return validateNumberWidths(resolved.OperandValues, 0xff)
		}
	}
	return nil
}

func validateNumberWidths(values []ast.Node, maximum uint64) error {
	for index, value := range values {
		if numberValue, ok := ast.NumberValue(value); ok && numberValue > maximum {
			return fmt.Errorf("operand %d value %d exceeds 0x%X", index, numberValue, maximum)
		}
	}
	return nil
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

func sameValues(left, right []ast.Node) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		leftValue, leftErr := formatValue(left[index])
		rightValue, rightErr := formatValue(right[index])
		if leftErr != nil || rightErr != nil || leftValue != rightValue {
			return false
		}
	}
	return true
}

func sameOperands(left, right []Operand) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		leftValue, leftErr := FormatOperand(left[index])
		rightValue, rightErr := FormatOperand(right[index])
		if leftErr != nil || rightErr != nil || leftValue != rightValue {
			return false
		}
	}
	return true
}

func formatOperandWithOptions(
	operand Operand,
	options FormatOptions,
	decimalValue bool,
	signedValue bool,
) (string, error) {

	switch operand.Kind {
	case OperandRegister:
		if !validDirectRegister(operand.Register) {
			return "", errInvalidOperandRegister
		}
		return formatRegisterName(operand.Register.String(), options), nil

	case OperandValue:
		if signedValue {
			return formatSignedOffset(operand.Value, options)
		}
		return formatValueWithOptions(operand.Value, options.MinimumHexDigits, decimalValue)

	case OperandIndirectRegister:
		if !validIndirectRegister(operand.Register) {
			return "", errInvalidOperandRegister
		}
		return "(" + formatRegisterName(operand.Register.String(), options) + ")", nil

	case OperandIndirectValue:
		value, err := formatValueWithOptions(operand.Value, options.MinimumHexDigits, false)
		if err != nil {
			return "", err
		}
		return "(" + value + ")", nil

	case OperandHLIncrement:
		return "(" + formatRegisterName(sm83.RegHL.String(), options) + "+)", nil

	case OperandHLDecrement:
		return "(" + formatRegisterName(sm83.RegHL.String(), options) + "-)", nil

	case OperandSPOffset:
		offset, err := formatSignedOffset(operand.Value, options)
		if err != nil {
			return "", err
		}
		if !strings.HasPrefix(offset, "-") {
			offset = "+" + offset
		}
		return formatRegisterName(sm83.RegSP.String(), options) + offset, nil

	default:
		return "", errInvalidOperandKind
	}
}

func formatSignedOffset(value ast.Node, options FormatOptions) (string, error) {
	numberValue, ok := ast.NumberValue(value)
	if !ok || numberValue > 0xff {
		return "", errInvalidOperandValue
	}
	negative := numberValue >= 0x80
	if negative {
		numberValue = 0x100 - numberValue
	}
	formatted, err := formatValueWithOptions(
		ast.NewNumber(numberValue),
		options.MinimumHexDigits,
		options.DecimalSignedOffsets,
	)
	if err != nil {
		return "", err
	}
	if negative {
		return "-" + formatted, nil
	}
	return formatted, nil
}

func formatRegisterName(name string, options FormatOptions) string {
	if options.Uppercase {
		return strings.ToUpper(name)
	}
	return name
}

func sm83PairRegisterOperand(operand Operand) bool {
	if operand.Kind != OperandRegister {
		return false
	}
	switch operand.Register {
	case sm83.RegAF, sm83.RegBC, sm83.RegDE, sm83.RegHL, sm83.RegSP:
		return true
	default:
		return false
	}
}

func formatValue(value ast.Node) (string, error) {
	return formatValueWithOptions(value, 0, false)
}

func formatValueWithOptions(value ast.Node, minimumHexDigits int, decimal bool) (string, error) {
	formatted, err := ast.FormatValue(value, ast.ValueFormatOptions{
		Decimal:          decimal,
		MinimumHexDigits: minimumHexDigits,
	})
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrUnsupportedValue, err)
	}
	return formatted, nil
}
