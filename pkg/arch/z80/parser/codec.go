package parser

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch/z80/profile"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
)

var (
	// ErrInvalidInstruction indicates inconsistent or incomplete typed Z80 instruction state.
	ErrInvalidInstruction = errors.New("invalid typed Z80 instruction")
	// ErrUnsupportedValue indicates a value node that the Z80 formatter cannot represent.
	ErrUnsupportedValue = errors.New("unsupported Z80 operand value")
)

// FormatOptions controls deterministic Z80 instruction spelling without
// changing the typed instruction or its encoding.
type FormatOptions struct {
	Indent                      string
	Uppercase                   bool
	MinimumHexDigits            int
	PairImmediateHexDigits      int
	DecimalBitIndexes           bool
	DecimalIndexedDisplacements bool
}

// BuildInstruction constructs and validates a typed Z80 instruction using the default profile.
func BuildInstruction(
	mnemonic string,
	variants []*cpuz80.Instruction,
	operands ...Operand,
) (ast.Instruction, error) {

	return BuildInstructionWithProfile(mnemonic, variants, profile.Default, operands...)
}

// BuildInstructionWithProfile constructs and validates a typed Z80 instruction without parsing text.
func BuildInstructionWithProfile(
	mnemonic string,
	variants []*cpuz80.Instruction,
	profileKind profile.Kind,
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
	resolved, err := resolveInstruction(variants, raw)
	if err != nil {
		return ast.Instruction{}, fmt.Errorf("%w: resolving '%s': %w", ErrInvalidInstruction, mnemonic, err)
	}
	if err := profile.ValidateInstruction(
		profileKind,
		resolved.Instruction,
		resolved.Addressing,
		resolved.RegisterParams,
	); err != nil {
		return ast.Instruction{}, fmt.Errorf("%w: validating profile '%s': %w", ErrInvalidInstruction, profileKind.String(), err)
	}
	if err := validateResolvedValues(*resolved); err != nil {
		return ast.Instruction{}, fmt.Errorf("%w: %w", ErrInvalidInstruction, err)
	}

	argument := ast.NewInstructionArgument(*resolved)
	return ast.NewInstruction(mnemonic, int(resolved.Addressing), argument, nil), nil
}

// ValidateInstruction checks typed Z80 metadata against its source-level operands and profile.
func ValidateInstruction(
	instruction ast.Instruction,
	variants []*cpuz80.Instruction,
	profileKind profile.Kind,
) error {

	resolved, err := resolvedArgument(instruction.Argument)
	if err != nil {
		return err
	}
	if err := validateResolvedMetadata(instruction, resolved, variants, profileKind); err != nil {
		return err
	}

	rebuilt, err := BuildInstructionWithProfile(instruction.Name, variants, profileKind, resolved.Operands...)
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
	if !sameValues(rebuiltResolved.OperandValues, resolved.OperandValues) {
		return fmt.Errorf("%w: resolved values do not match operands", ErrInvalidInstruction)
	}
	return nil
}

// FormatInstruction returns one deterministic, parseable Z80 instruction line.
func FormatInstruction(instruction ast.Instruction) (string, error) {
	return FormatInstructionWithOptions(instruction, FormatOptions{})
}

// FormatInstructionWithOptions returns one deterministic, parseable Z80
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
			(strings.EqualFold(mnemonic, cpuz80.BitName) ||
				strings.EqualFold(mnemonic, cpuz80.SetName) ||
				strings.EqualFold(mnemonic, cpuz80.ResName))
		if index == 1 && strings.EqualFold(mnemonic, cpuz80.LdName) &&
			len(resolved.Operands) == 2 && z80PairRegisterOperand(resolved.Operands[0]) {

			operandOptions.MinimumHexDigits = options.PairImmediateHexDigits
		}
		formatted[index], err = formatOperandWithOptions(operand, operandOptions, decimalValue)
		if err != nil {
			return "", fmt.Errorf("formatting operand %d: %w", index, err)
		}
	}
	return options.Indent + mnemonic + " " + strings.Join(formatted, ","), nil
}

// FormatOperand returns the canonical Z80 spelling for one typed operand.
// It lets downstream typed consumers retain symbolic operand shapes without
// formatting and splitting a complete instruction.
func FormatOperand(operand Operand) (string, error) {
	return formatOperandWithOptions(operand, FormatOptions{}, false)
}

func validateResolvedMetadata(
	instruction ast.Instruction,
	resolved ResolvedInstruction,
	variants []*cpuz80.Instruction,
	profileKind profile.Kind,
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
	if err := profile.ValidateInstruction(
		profileKind,
		resolved.Instruction,
		resolved.Addressing,
		resolved.RegisterParams,
	); err != nil {
		return fmt.Errorf("%w: validating profile '%s': %w", ErrInvalidInstruction, profileKind.String(), err)
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

	if isBitOperation(resolved.Instruction) && len(resolved.OperandValues) > 0 {
		if value, ok := ast.NumberValue(resolved.OperandValues[0]); ok && value > 7 {
			return fmt.Errorf("bit number %d exceeds 7", value)
		}
	}
	for _, operand := range resolved.Operands {
		if operand.Kind != OperandIndexed {
			continue
		}
		if value, ok := ast.NumberValue(operand.Value); ok && value > 0xff {
			return fmt.Errorf("indexed displacement %d exceeds byte", value)
		}
	}

	baseWidth := 1
	if opcodeInfo.Prefix != 0 {
		baseWidth++
	}
	remaining := int(opcodeInfo.Size) - baseWidth
	switch addressing {
	case cpuz80.ImmediateAddressing:
		if len(resolved.OperandValues) >= 2 {
			return validateNumberWidths(resolved.OperandValues, 0xff)
		}
		if remaining == 1 {
			return validateNumberWidths(resolved.OperandValues, 0xff)
		}
		if remaining == 2 {
			return validateNumberWidths(resolved.OperandValues, 0xffff)
		}
	case cpuz80.ExtendedAddressing, cpuz80.RelativeAddressing:
		return validateNumberWidths(resolved.OperandValues, 0xffff)
	case cpuz80.RegisterIndirectAddressing, cpuz80.PortAddressing:
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

func formatOperandWithOptions(
	operand Operand,
	options FormatOptions,
	decimalValue bool,
) (string, error) {

	switch operand.Kind {
	case OperandRegister:
		if !validRegisterParam(operand.Register) {
			return "", errInvalidOperandRegister
		}
		return formatRegisterName(operand.Register.String(), options), nil

	case OperandValue:
		return formatValueWithOptions(operand.Value, options.MinimumHexDigits, decimalValue)

	case OperandIndirectRegister:
		register := sourceRegisterParam(operand.Register)
		if !validRegisterParam(register) {
			return "", errInvalidOperandRegister
		}
		return "(" + formatRegisterName(register.String(), options) + ")", nil

	case OperandIndirectValue:
		value, err := formatValueWithOptions(operand.Value, options.MinimumHexDigits, false)
		if err != nil {
			return "", err
		}
		return "(" + value + ")", nil

	case OperandIndexed:
		return formatIndexedOperand(operand, options)

	default:
		return "", errInvalidOperandKind
	}
}

func formatIndexedOperand(operand Operand, options FormatOptions) (string, error) {
	register := indexedRegisterParam(operand.Register)
	if register == cpuz80.RegNone {
		return "", errInvalidOperandRegister
	}
	name := formatRegisterName(strings.Trim(register.String(), "()"), options)
	if value, ok := ast.NumberValue(operand.Value); ok {
		if options.DecimalIndexedDisplacements {
			if value >= 0x80 && value <= 0xff {
				return fmt.Sprintf("(%s-%d)", name, 0x100-value), nil
			}
			return fmt.Sprintf("(%s+%d)", name, value), nil
		}
		if value >= 0x80 && value <= 0xff {
			return fmt.Sprintf("(%s-0x%X)", name, 0x100-value), nil
		}
		return fmt.Sprintf("(%s+0x%X)", name, value), nil
	}
	value, err := formatValue(operand.Value)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(value, "0x0-(") && strings.HasSuffix(value, ")") {
		return "(" + name + " - " + strings.TrimSuffix(strings.TrimPrefix(value, "0x0-("), ")") + ")", nil
	}
	return "(" + name + "+" + value + ")", nil
}

func formatRegisterName(name string, options FormatOptions) string {
	if options.Uppercase {
		return strings.ToUpper(name)
	}
	return name
}

func z80PairRegisterOperand(operand Operand) bool {
	if operand.Kind != OperandRegister {
		return false
	}
	switch operand.Register {
	case cpuz80.RegBC, cpuz80.RegDE, cpuz80.RegHL, cpuz80.RegIX, cpuz80.RegIY, cpuz80.RegSP:
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
