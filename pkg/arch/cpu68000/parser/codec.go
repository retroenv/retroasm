package parser

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu68000"
)

var ErrInvalidInstruction = errors.New("invalid typed CPU68000 instruction")

var canonicalConditions = [...]string{
	"t", "f", "hi", "ls", "cc", "cs", "ne", "eq",
	"vc", "vs", "pl", "mi", "ge", "lt", "gt", "le",
}

// BuildInstruction constructs a typed CPU68000 instruction without parsing text.
func BuildInstruction(
	mnemonic string,
	instruction *cpu68000.Instruction,
	operands Operands,
) (ast.Instruction, error) {

	mnemonic = strings.TrimSpace(mnemonic)
	if mnemonic == "" || instruction == nil {
		return ast.Instruction{}, fmt.Errorf("%w: mnemonic %q", ErrInvalidInstruction, mnemonic)
	}

	baseMnemonic, suffixSize := ParseSizeSuffix(mnemonic)
	_, condition, hasCondition := ParseConditionCode(baseMnemonic)
	size := operands.Size
	if suffixSize != 0 {
		if size != 0 && size != suffixSize {
			return ast.Instruction{}, fmt.Errorf("%w: conflicting operand sizes", ErrInvalidInstruction)
		}
		size = suffixSize
	}
	if size == 0 {
		size = cpu68000.SizeWord
	}
	if instruction.Name == cpu68000.MOVEQName {
		size = cpu68000.SizeLong
	}

	resolved := ResolvedInstruction{
		Instruction: instruction,
		Size:        size,
		SrcEA:       copyEffectiveAddress(operands.Source),
		DstEA:       copyEffectiveAddress(operands.Destination),
		Extra:       operands.Extra,
	}
	if hasCondition {
		resolved.Extra = condition
	}
	resolveSpecialOperandModes(&resolved)
	if err := validateResolved(resolved); err != nil {
		return ast.Instruction{}, err
	}

	argument := ast.NewInstructionArgument(resolved)
	return ast.NewInstruction(instruction.Name, int(cpu68000.NoAddressing), argument, nil), nil
}

// ValidateInstruction checks typed CPU68000 metadata and effective addresses.
func ValidateInstruction(instruction ast.Instruction, expected *cpu68000.Instruction) error {
	resolved, err := resolvedArgument(instruction.Argument)
	if err != nil {
		return err
	}
	if expected == nil || resolved.Instruction != expected {
		return fmt.Errorf("%w: retained instruction does not match mnemonic", ErrInvalidInstruction)
	}
	if instruction.Addressing != int(cpu68000.NoAddressing) {
		return fmt.Errorf("%w: unexpected AST addressing %d", ErrInvalidInstruction, instruction.Addressing)
	}
	if len(instruction.Modifier) != 0 {
		return fmt.Errorf("%w: instruction modifiers are not supported", ErrInvalidInstruction)
	}
	return validateResolved(resolved)
}

// FormatOptions controls deterministic CPU68000 instruction spelling.
type FormatOptions struct {
	Indent    string
	Uppercase bool
}

// FormatInstruction returns one deterministic, parseable CPU68000 instruction line.
func FormatInstruction(instruction ast.Instruction) (string, error) {
	return FormatInstructionWithOptions(instruction, FormatOptions{})
}

// FormatInstructionWithOptions formats one typed CPU68000 instruction.
func FormatInstructionWithOptions(instruction ast.Instruction, options FormatOptions) (string, error) {
	resolved, err := resolvedArgument(instruction.Argument)
	if err != nil {
		return "", err
	}
	mnemonic, err := formatMnemonic(resolved)
	if err != nil {
		return "", err
	}
	if options.Uppercase {
		mnemonic = strings.ToUpper(mnemonic)
	}

	operands, err := formattedOperands(resolved, options)
	if err != nil {
		return "", err
	}
	if len(operands) == 0 {
		return options.Indent + mnemonic, nil
	}
	return options.Indent + mnemonic + " " + strings.Join(operands, ","), nil
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

func resolveSpecialOperandModes(resolved *ResolvedInstruction) {
	if resolved.Instruction.Name == cpu68000.ADDQName || resolved.Instruction.Name == cpu68000.SUBQName {
		if resolved.SrcEA != nil && resolved.SrcEA.Mode == cpu68000.ImmediateMode {
			resolved.SrcEA.Mode = cpu68000.QuickImmediateMode
		}
	}
	if resolved.Instruction.Name != cpu68000.MOVEMName {
		return
	}
	if resolved.SrcEA != nil && resolved.SrcEA.RegList != 0 {
		resolved.Extra = 0
	}
	if resolved.DstEA != nil && resolved.DstEA.RegList != 0 {
		resolved.Extra = 1
	}
}

func validateResolved(resolved ResolvedInstruction) error {
	if resolved.Instruction == nil {
		return fmt.Errorf("%w: missing instruction metadata", ErrInvalidInstruction)
	}
	if resolved.Size != cpu68000.SizeByte && resolved.Size != cpu68000.SizeWord && resolved.Size != cpu68000.SizeLong {
		return fmt.Errorf("%w: operand size %d", ErrInvalidInstruction, resolved.Size)
	}
	if resolved.Extra > 15 && isConditionalInstruction(resolved.Instruction.Name) {
		return fmt.Errorf("%w: condition code %d", ErrInvalidInstruction, resolved.Extra)
	}
	if err := validateOperandShape(resolved); err != nil {
		return err
	}
	if err := validateEffectiveAddress(resolved.SrcEA, resolved.Size); err != nil {
		return fmt.Errorf("%w: source: %w", ErrInvalidInstruction, err)
	}
	if err := validateEffectiveAddress(resolved.DstEA, resolved.Size); err != nil {
		return fmt.Errorf("%w: destination: %w", ErrInvalidInstruction, err)
	}
	return validateSpecialValues(resolved)
}

func validateOperandShape(resolved ResolvedInstruction) error {
	name := resolved.Instruction.Name
	source, destination := resolved.SrcEA != nil, resolved.DstEA != nil
	switch {
	case isNoOperandInstruction(name):
		if source || destination {
			return fmt.Errorf("%w: %s takes no operands", ErrInvalidInstruction, name)
		}
	case name == cpu68000.TRAPName || name == cpu68000.STOPName:
		if !source || destination {
			return fmt.Errorf("%w: %s requires one source operand", ErrInvalidInstruction, name)
		}
	case isDestinationOnlyInstruction(name), isBranchInstruction(name):
		if source || !destination {
			return fmt.Errorf("%w: %s requires one destination operand", ErrInvalidInstruction, name)
		}
	case isShiftRotateInstruction(name):
		if !destination {
			return fmt.Errorf("%w: %s requires a destination operand", ErrInvalidInstruction, name)
		}
	default:
		if !source || !destination {
			return fmt.Errorf("%w: %s requires source and destination operands", ErrInvalidInstruction, name)
		}
	}
	return nil
}

func isDestinationOnlyInstruction(name string) bool {
	switch name {
	case cpu68000.CLRName, cpu68000.NEGName, cpu68000.NEGXName, cpu68000.NOTName,
		cpu68000.TSTName, cpu68000.NBCDName, cpu68000.TASName, cpu68000.PEAName,
		cpu68000.JMPName, cpu68000.JSRName, cpu68000.SWAPName, cpu68000.EXTName,
		cpu68000.UNLKName, cpu68000.SccName:
		return true
	default:
		return false
	}
}

func isShiftRotateInstruction(name string) bool {
	switch name {
	case cpu68000.ASLName, cpu68000.ASRName, cpu68000.LSLName, cpu68000.LSRName,
		cpu68000.ROLName, cpu68000.RORName, cpu68000.ROXLName, cpu68000.ROXRName:
		return true
	default:
		return false
	}
}

func isConditionalInstruction(name string) bool {
	return name == cpu68000.BccName || name == cpu68000.DBccName || name == cpu68000.SccName
}

//nolint:cyclop // validation mirrors the complete effective-address mode set
func validateEffectiveAddress(address *EffectiveAddress, size cpu68000.OperandSize) error {
	if address == nil {
		return nil
	}
	if address.RegList != 0 {
		if address.Mode != cpu68000.NoAddressing || address.Value != nil {
			return errors.New("register list has incompatible effective-address fields")
		}
		return nil
	}

	switch address.Mode {
	case cpu68000.DataRegDirectMode:
		return validateRegister(address.Register)
	case cpu68000.AddrRegDirectMode:
		if address.Register == regUSP {
			return nil
		}
		return validateRegister(address.Register)
	case cpu68000.AddrRegIndirectMode, cpu68000.PostIncrementMode, cpu68000.PreDecrementMode:
		return validateRegister(address.Register)
	case cpu68000.DisplacementMode, cpu68000.PCDisplacementMode:
		return validateValue(address.Value, math.MaxUint16)
	case cpu68000.IndexedMode, cpu68000.PCIndexedMode:
		if address.IndexSize != cpu68000.SizeWord && address.IndexSize != cpu68000.SizeLong {
			return fmt.Errorf("invalid index size %d", address.IndexSize)
		}
		if err := validateRegister(address.IndexReg); err != nil {
			return err
		}
		return validateValue(address.Value, math.MaxUint8)
	case cpu68000.AbsShortMode:
		return validateValue(address.Value, math.MaxUint16)
	case cpu68000.AbsLongMode:
		return validateValue(address.Value, math.MaxUint32)
	case cpu68000.ImmediateMode:
		return validateValue(address.Value, sizeMaximum(size))
	case cpu68000.QuickImmediateMode:
		return validateValue(address.Value, math.MaxUint8)
	case cpu68000.StatusRegMode:
		if address.Register != regSR && address.Register != regCCR {
			return fmt.Errorf("invalid status register %d", address.Register)
		}
		return nil
	default:
		return fmt.Errorf("unsupported effective-address mode %d", address.Mode)
	}
}

func validateRegister(register uint8) error {
	if register > 7 {
		return fmt.Errorf("register %d exceeds 7", register)
	}
	return nil
}

func validateValue(value ast.Node, maximum uint64) error {
	if value == nil {
		return errors.New("value is missing")
	}
	if numberValue, ok := ast.NumberValue(value); ok && numberValue > maximum {
		return fmt.Errorf("value %d exceeds 0x%X", numberValue, maximum)
	}
	return nil
}

func sizeMaximum(size cpu68000.OperandSize) uint64 {
	switch size {
	case cpu68000.SizeByte:
		return math.MaxUint8
	case cpu68000.SizeWord:
		return math.MaxUint16
	default:
		return math.MaxUint32
	}
}

func validateSpecialValues(resolved ResolvedInstruction) error {
	switch resolved.Instruction.Name {
	case cpu68000.ADDQName, cpu68000.SUBQName:
		if value, ok := ast.NumberValue(resolved.SrcEA.Value); !ok || value < 1 || value > 8 {
			return fmt.Errorf("%w: quick value must be 1..8", ErrInvalidInstruction)
		}
	case cpu68000.TRAPName:
		if value, ok := ast.NumberValue(resolved.SrcEA.Value); !ok || value > 15 {
			return fmt.Errorf("%w: trap vector must be 0..15", ErrInvalidInstruction)
		}
	case cpu68000.MOVEMName:
		sourceList := resolved.SrcEA.RegList != 0
		destinationList := resolved.DstEA.RegList != 0
		if sourceList == destinationList {
			return fmt.Errorf("%w: MOVEM requires one register list", ErrInvalidInstruction)
		}
	}
	return nil
}

func formatMnemonic(resolved ResolvedInstruction) (string, error) {
	name := strings.ToLower(resolved.Instruction.Name)
	if isConditionalInstruction(resolved.Instruction.Name) {
		if int(resolved.Extra) >= len(canonicalConditions) {
			return "", fmt.Errorf("%w: condition code %d", ErrInvalidInstruction, resolved.Extra)
		}
		condition := canonicalConditions[resolved.Extra]
		switch resolved.Instruction.Name {
		case cpu68000.BccName:
			name = "b" + condition
		case cpu68000.DBccName:
			name = "db" + condition
		case cpu68000.SccName:
			name = "s" + condition
		}
	}
	if instructionUsesSizeSuffix(resolved.Instruction.Name) {
		name += sizeSuffix(resolved.Size)
	}
	return name, nil
}

func instructionUsesSizeSuffix(name string) bool {
	if isNoOperandInstruction(name) || name == cpu68000.MOVEQName || name == cpu68000.TRAPName ||
		name == cpu68000.STOPName || name == cpu68000.LINKName || name == cpu68000.UNLKName ||
		name == cpu68000.SWAPName || name == cpu68000.EXGName || name == cpu68000.JMPName ||
		name == cpu68000.JSRName || name == cpu68000.PEAName || name == cpu68000.SccName ||
		name == cpu68000.DBccName {

		return false
	}
	return true
}

func sizeSuffix(size cpu68000.OperandSize) string {
	switch size {
	case cpu68000.SizeByte:
		return ".b"
	case cpu68000.SizeLong:
		return ".l"
	default:
		return ".w"
	}
}

func formattedOperands(resolved ResolvedInstruction, options FormatOptions) ([]string, error) {
	if resolved.Instruction.Name == cpu68000.MOVEMName {
		return formatMOVEMOperands(resolved, options)
	}
	var addresses []*EffectiveAddress
	if resolved.SrcEA != nil {
		addresses = append(addresses, resolved.SrcEA)
	}
	if resolved.DstEA != nil {
		addresses = append(addresses, resolved.DstEA)
	}
	formatted := make([]string, len(addresses))
	for index, address := range addresses {
		value, err := formatEffectiveAddress(address, resolved.Size, options)
		if err != nil {
			return nil, fmt.Errorf("formatting operand %d: %w", index, err)
		}
		formatted[index] = value
	}
	return formatted, nil
}

func formatMOVEMOperands(resolved ResolvedInstruction, options FormatOptions) ([]string, error) {
	if resolved.Extra == 0 {
		list, err := formatEffectiveAddress(resolved.SrcEA, resolved.Size, options)
		if err != nil {
			return nil, err
		}
		destination, err := formatEffectiveAddress(resolved.DstEA, resolved.Size, options)
		if err != nil {
			return nil, err
		}
		return []string{list, destination}, nil
	}
	source, err := formatEffectiveAddress(resolved.SrcEA, resolved.Size, options)
	if err != nil {
		return nil, err
	}
	list, err := formatEffectiveAddress(resolved.DstEA, resolved.Size, options)
	if err != nil {
		return nil, err
	}
	return []string{source, list}, nil
}

//nolint:cyclop,funlen // formatting mirrors the complete effective-address mode set
func formatEffectiveAddress(
	address *EffectiveAddress,
	operandSize cpu68000.OperandSize,
	options FormatOptions,
) (string, error) {

	if address == nil {
		return "", errors.New("effective address is missing")
	}
	if address.RegList != 0 {
		return formatRegisterList(address.RegList, options), nil
	}
	register := func(prefix string, number uint8) string {
		name := fmt.Sprintf("%s%d", prefix, number)
		if options.Uppercase {
			return strings.ToUpper(name)
		}
		return name
	}
	addressRegister := register("a", address.Register)

	switch address.Mode {
	case cpu68000.DataRegDirectMode:
		return register("d", address.Register), nil
	case cpu68000.AddrRegDirectMode:
		if address.Register == regUSP {
			return formatKeyword("usp", options), nil
		}
		return addressRegister, nil
	case cpu68000.AddrRegIndirectMode:
		return "(" + addressRegister + ")", nil
	case cpu68000.PostIncrementMode:
		return "(" + addressRegister + ")+", nil
	case cpu68000.PreDecrementMode:
		return "-(" + addressRegister + ")", nil
	case cpu68000.DisplacementMode:
		return formatDisplacement(address.Value, addressRegister, address.Negative)
	case cpu68000.IndexedMode:
		return formatIndexedAddress(address, addressRegister, options)
	case cpu68000.AbsShortMode:
		return formatAbsoluteAddress(address.Value, ".w", math.MaxUint16, address.Negative)
	case cpu68000.AbsLongMode:
		return formatAbsoluteAddress(address.Value, ".l", math.MaxUint32, address.Negative)
	case cpu68000.PCDisplacementMode:
		return formatDisplacement(address.Value, formatKeyword("pc", options), address.Negative)
	case cpu68000.PCIndexedMode:
		return formatIndexedAddress(address, formatKeyword("pc", options), options)
	case cpu68000.ImmediateMode:
		value, err := format68000Value(address.Value, sizeMaximum(operandSize))
		if err != nil {
			return "", err
		}
		return "#" + signedCPU68000Value(value, address.Negative), nil
	case cpu68000.QuickImmediateMode:
		value, err := format68000Value(address.Value, math.MaxUint8)
		if err != nil {
			return "", err
		}
		return "#" + signedCPU68000Value(value, address.Negative), nil
	case cpu68000.StatusRegMode:
		if address.Register == regCCR {
			return formatKeyword("ccr", options), nil
		}
		return formatKeyword("sr", options), nil
	default:
		return "", fmt.Errorf("unsupported effective-address mode %d", address.Mode)
	}
}

func formatDisplacement(value ast.Node, base string, negative bool) (string, error) {
	formatted, err := format68000Value(value, math.MaxUint16)
	if err != nil {
		return "", err
	}
	return signedCPU68000Value(formatted, negative) + "(" + base + ")", nil
}

func formatIndexedAddress(address *EffectiveAddress, base string, options FormatOptions) (string, error) {
	formatted, err := format68000Value(address.Value, math.MaxUint8)
	if err != nil {
		return "", err
	}
	prefix := "d"
	if address.IsAddrReg {
		prefix = "a"
	}
	index := fmt.Sprintf("%s%d%s", prefix, address.IndexReg, sizeSuffix(address.IndexSize))
	if options.Uppercase {
		index = strings.ToUpper(index)
	}
	return signedCPU68000Value(formatted, address.Negative) + "(" + base + "," + index + ")", nil
}

func formatAbsoluteAddress(value ast.Node, suffix string, maximum uint64, negative bool) (string, error) {
	formatted, err := format68000Value(value, maximum)
	if err != nil {
		return "", err
	}
	return signedCPU68000Value(formatted, negative) + suffix, nil
}

func signedCPU68000Value(value string, negative bool) string {
	if negative {
		return "-" + value
	}
	return value
}

func format68000Value(value ast.Node, maximum uint64) (string, error) {
	digits := 0
	if _, ok := ast.NumberValue(value); ok {
		switch maximum {
		case math.MaxUint8:
			digits = 2
		case math.MaxUint16:
			digits = 4
		case math.MaxUint32:
			digits = 8
		}
	}
	formatted, err := ast.FormatValue(value, ast.ValueFormatOptions{MinimumHexDigits: digits})
	if err != nil {
		return "", err //nolint:wrapcheck // shared AST formatter identifies unsupported values
	}
	return formatted, nil
}

func formatRegisterList(mask uint16, options FormatOptions) string {
	registers := make([]string, 0, 16)
	for bit := range 16 {
		if mask&(1<<bit) == 0 {
			continue
		}
		prefix := "d"
		number := bit
		if bit >= 8 {
			prefix = "a"
			number -= 8
		}
		name := fmt.Sprintf("%s%d", prefix, number)
		if options.Uppercase {
			name = strings.ToUpper(name)
		}
		registers = append(registers, name)
	}
	return strings.Join(registers, "/")
}

func formatKeyword(keyword string, options FormatOptions) string {
	if options.Uppercase {
		return strings.ToUpper(keyword)
	}
	return keyword
}
