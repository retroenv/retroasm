package parser

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/retroenv/retroasm/pkg/number"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu65816"
)

var (
	// ErrInvalidInstruction indicates inconsistent typed CPU65816 instruction state.
	ErrInvalidInstruction = errors.New("invalid typed CPU65816 instruction")
	// ErrUnsupportedAddressing indicates an operand shape unavailable for a mnemonic.
	ErrUnsupportedAddressing = errors.New("unsupported CPU65816 addressing")
)

var addressingOperandShapes = map[cpu65816.AddressingMode]operandShapeInfo{
	cpu65816.ImmediateAddressing:                      {OperandImmediate, AddressDefault},
	cpu65816.DirectPageAddressing:                     {OperandAddress, AddressDirectPage},
	cpu65816.AbsoluteAddressing:                       {OperandAddress, AddressAbsolute},
	cpu65816.AbsoluteLongAddressing:                   {OperandAddress, AddressLong},
	cpu65816.RelativeAddressing:                       {OperandAddress, AddressDefault},
	cpu65816.RelativeLongAddressing:                   {OperandAddress, AddressDefault},
	AbsoluteDirectPageAddressing:                      {OperandAddress, AddressDefault},
	cpu65816.DirectPageIndexedXAddressing:             {OperandIndexedX, AddressDirectPage},
	cpu65816.AbsoluteIndexedXAddressing:               {OperandIndexedX, AddressAbsolute},
	cpu65816.AbsoluteLongIndexedXAddressing:           {OperandIndexedX, AddressLong},
	XAddressing:                                       {OperandIndexedX, AddressDefault},
	cpu65816.DirectPageIndexedYAddressing:             {OperandIndexedY, AddressDirectPage},
	cpu65816.AbsoluteIndexedYAddressing:               {OperandIndexedY, AddressAbsolute},
	YAddressing:                                       {OperandIndexedY, AddressDefault},
	cpu65816.DirectPageIndirectAddressing:             {OperandIndirect, AddressDirectPage},
	cpu65816.AbsoluteIndirectAddressing:               {OperandIndirect, AddressAbsolute},
	cpu65816.DirectPageIndexedXIndirectAddressing:     {OperandIndexedXIndirect, AddressDirectPage},
	cpu65816.AbsoluteIndexedXIndirectAddressing:       {OperandIndexedXIndirect, AddressAbsolute},
	cpu65816.DirectPageIndirectIndexedYAddressing:     {OperandIndirectIndexedY, AddressDirectPage},
	cpu65816.DirectPageIndirectLongAddressing:         {OperandIndirectLong, AddressDirectPage},
	cpu65816.AbsoluteIndirectLongAddressing:           {OperandIndirectLong, AddressAbsolute},
	cpu65816.DirectPageIndirectLongIndexedYAddressing: {OperandIndirectLongIndexedY, AddressDirectPage},
	cpu65816.StackRelativeAddressing:                  {OperandStackRelative, AddressDirectPage},
	cpu65816.StackRelativeIndirectIndexedYAddressing:  {OperandStackRelativeIndirectIndexedY, AddressDirectPage},
}

// ResolvedInstruction retains the selected instruction, operands, and M/X state.
type ResolvedInstruction struct {
	Instruction *cpu65816.Instruction
	Addressing  cpu65816.AddressingMode
	Operands    Operands
	State       State
}

// CopyInstructionArgument returns a deep copy suitable for AST duplication.
func (resolved ResolvedInstruction) CopyInstructionArgument() any {
	resolved.Operands = copyOperands(resolved.Operands)
	return resolved
}

// InstructionFormKey returns the selected addressing, processor state, and operand shapes.
func (resolved ResolvedInstruction) InstructionFormKey() string {
	operands := make([]string, len(resolved.Operands))
	for index, operand := range resolved.Operands {
		modifiers := make([]string, len(operand.Modifiers))
		for modifierIndex, modifier := range operand.Modifiers {
			modifiers[modifierIndex] = modifier.Operator.Operator
		}
		operands[index] = fmt.Sprintf(
			"%d/%d/%s/%s",
			operand.Kind,
			operand.Size,
			strings.Join(modifiers, ""),
			ast.InstructionArgumentForm(operand.Value),
		)
	}
	return fmt.Sprintf(
		"addressing=%d;state=%d/%d/%d/%d;operands=%s",
		resolved.Addressing,
		resolved.State.AccumulatorWidth,
		resolved.State.IndexWidth,
		resolved.State.Carry,
		resolved.State.Emulation,
		strings.Join(operands, ","),
	)
}

// InstructionStateTransitionForm returns the state change made by this instruction.
func (resolved ResolvedInstruction) InstructionStateTransitionForm() (string, bool) {
	if resolved.Instruction == nil {
		return "", false
	}

	next := nextState(resolved.State, resolved.Instruction, resolved.Operands)
	if next == resolved.State {
		return "", false
	}
	return fmt.Sprintf(
		"%s:%s->%s",
		resolved.Instruction.Name,
		instructionStateForm(resolved.State),
		instructionStateForm(next),
	), true
}

// OpcodeInfo returns encoding metadata for one concrete addressing mode.
func (resolved ResolvedInstruction) OpcodeInfo(addressing cpu65816.AddressingMode) (cpu65816.OpcodeInfo, error) {
	if resolved.Instruction == nil {
		return cpu65816.OpcodeInfo{}, fmt.Errorf("%w: missing instruction metadata", ErrInvalidInstruction)
	}
	info, ok := resolved.Instruction.Addressing[addressing]
	if !ok {
		return cpu65816.OpcodeInfo{}, fmt.Errorf(
			"%w: %s mode %d",
			ErrUnsupportedAddressing,
			resolved.Instruction.Name,
			addressing,
		)
	}
	return info, nil
}

// EncodedSize returns the instruction size for its retained M/X state.
func (resolved ResolvedInstruction) EncodedSize(addressing cpu65816.AddressingMode) (int, error) {
	info, err := resolved.OpcodeInfo(addressing)
	if err != nil {
		return 0, err
	}
	size := int(info.BaseSize)
	if addressing != cpu65816.ImmediateAddressing {
		return size, nil
	}
	width := immediateWidth(resolved.Instruction, resolved.State)
	if width == WidthUnknown {
		return 0, fmt.Errorf("%w: %s immediate width is runtime-dependent", ErrInvalidInstruction, resolved.Instruction.Name)
	}
	if width == WidthWord {
		size++
	}
	return size, nil
}

// InstructionReferences returns copied symbol-bearing operand views for stream validation.
func (resolved ResolvedInstruction) InstructionReferences() []ast.InstructionReference {
	references := make([]ast.InstructionReference, 0, len(resolved.Operands))

	for _, operand := range resolved.Operands {
		if !operandReferencesSymbol(operand.Value) {
			continue
		}
		references = append(references, ast.InstructionReference{
			Value:         operand.Value.Copy(),
			Modifiers:     slices.Clone(operand.Modifiers),
			ReferenceType: ast.FullAddress,
		})
	}
	return references
}

type operandShapeInfo struct {
	kind OperandKind
	size AddressSize
}

func instructionStateForm(state State) string {
	return fmt.Sprintf(
		"%d/%d/%d/%d",
		state.AccumulatorWidth,
		state.IndexWidth,
		state.Carry,
		state.Emulation,
	)
}

func operandReferencesSymbol(value ast.Node) bool {
	if value == nil {
		return false
	}
	if ast.SymbolName(value) != "" {
		return true
	}
	expression, ok := value.(ast.Expression)
	if !ok {
		return false
	}
	_, _, ok = ast.ParseSymbolReference(expression.Value)
	return ok
}

func resolvedFromParsed(
	instruction ast.Instruction,
	details *cpu65816.Instruction,
	state State,
) (ast.Instruction, ResolvedInstruction, error) {

	operands, err := operandsFromParsed(instruction)
	if err != nil {
		return ast.Instruction{}, ResolvedInstruction{}, err
	}
	resolved := ResolvedInstruction{
		Instruction: details,
		Addressing:  cpu65816.AddressingMode(instruction.Addressing),
		Operands:    operands,
		State:       state,
	}
	if err := validateResolved(resolved); err != nil {
		return ast.Instruction{}, ResolvedInstruction{}, err
	}
	instruction.Argument = ast.NewInstructionArgument(resolved)
	instruction.Modifier = nil
	return instruction, resolved, nil
}

func operandsFromParsed(instruction ast.Instruction) (Operands, error) {
	addressing := cpu65816.AddressingMode(instruction.Addressing)
	if addressing == cpu65816.ImpliedAddressing {
		return nil, nil
	}
	if addressing == cpu65816.AccumulatorAddressing {
		return Operands{AccumulatorOperand()}, nil
	}
	if addressing == cpu65816.BlockMoveAddressing {
		packed, ok := ast.NumberValue(instruction.Argument)
		if !ok {
			return nil, fmt.Errorf("%w: block-move banks are not numeric", ErrInvalidInstruction)
		}
		return BlockMoveOperands(ast.NewNumber(packed>>8), ast.NewNumber(packed&math.MaxUint8)), nil
	}
	if instruction.Argument == nil {
		return nil, fmt.Errorf("%w: missing operand", ErrInvalidInstruction)
	}

	kind, size, err := operandShape(addressing)
	if err != nil {
		return nil, err
	}
	operand := Operand{
		Kind:      kind,
		Size:      size,
		Value:     instruction.Argument.Copy(),
		Modifiers: slices.Clone(instruction.Modifier),
	}
	return Operands{operand}, nil
}

func operandShape(addressing cpu65816.AddressingMode) (OperandKind, AddressSize, error) {
	shape, ok := addressingOperandShapes[addressing]
	if !ok {
		return OperandInvalid, AddressDefault, fmt.Errorf("%w: mode %d", ErrUnsupportedAddressing, addressing)
	}
	return shape.kind, shape.size, nil
}

func resolveOperands(instruction *cpu65816.Instruction, operands Operands) (cpu65816.AddressingMode, error) {
	switch len(operands) {
	case 0:
		if instruction.HasAddressing(cpu65816.ImpliedAddressing) {
			return cpu65816.ImpliedAddressing, nil
		}
		return cpu65816.NoAddressing, ErrUnsupportedAddressing
	case 1:
		return resolveSingleOperand(instruction, operands[0])
	case 2:
		if instruction.HasAddressing(cpu65816.BlockMoveAddressing) &&
			operands[0].Kind == OperandBlockMoveBank && operands[1].Kind == OperandBlockMoveBank {

			return cpu65816.BlockMoveAddressing, nil
		}
	}
	return cpu65816.NoAddressing, ErrUnsupportedAddressing
}

func resolveSingleOperand(instruction *cpu65816.Instruction, operand Operand) (cpu65816.AddressingMode, error) {
	var addressing cpu65816.AddressingMode
	switch operand.Kind {
	case OperandAccumulator:
		addressing = cpu65816.AccumulatorAddressing
	case OperandImmediate:
		addressing = cpu65816.ImmediateAddressing
	case OperandAddress:
		addressing = resolveAddress(instruction, operand)
	case OperandIndexedX:
		addressing = resolveIndexedX(instruction, operand)
	case OperandIndexedY:
		addressing = resolveIndexedY(instruction, operand)
	case OperandIndirect:
		addressing = resolveSizedPair(instruction, operand, cpu65816.DirectPageIndirectAddressing, cpu65816.AbsoluteIndirectAddressing)
	case OperandIndexedXIndirect:
		addressing = resolveSizedPair(
			instruction,
			operand,
			cpu65816.DirectPageIndexedXIndirectAddressing,
			cpu65816.AbsoluteIndexedXIndirectAddressing,
		)
	case OperandIndirectIndexedY:
		addressing = cpu65816.DirectPageIndirectIndexedYAddressing
	case OperandIndirectLong:
		addressing = resolveSizedPair(
			instruction,
			operand,
			cpu65816.DirectPageIndirectLongAddressing,
			cpu65816.AbsoluteIndirectLongAddressing,
		)
	case OperandIndirectLongIndexedY:
		addressing = cpu65816.DirectPageIndirectLongIndexedYAddressing
	case OperandStackRelative:
		addressing = cpu65816.StackRelativeAddressing
	case OperandStackRelativeIndirectIndexedY:
		addressing = cpu65816.StackRelativeIndirectIndexedYAddressing
	default:
		return cpu65816.NoAddressing, ErrUnsupportedAddressing
	}
	if !supportsAddressing(instruction, addressing) {
		return cpu65816.NoAddressing, ErrUnsupportedAddressing
	}
	return addressing, nil
}

func resolveAddress(instruction *cpu65816.Instruction, operand Operand) cpu65816.AddressingMode {
	if operand.Size != AddressDefault {
		return sizedAddressing(
			operand.Size,
			cpu65816.DirectPageAddressing,
			cpu65816.AbsoluteAddressing,
			cpu65816.AbsoluteLongAddressing,
		)
	}
	for _, addressing := range []cpu65816.AddressingMode{
		cpu65816.RelativeAddressing,
		cpu65816.RelativeLongAddressing,
	} {
		if instruction.HasAddressing(addressing) {
			return addressing
		}
	}
	return resolveDefaultAddress(instruction, operand.Value)
}

func resolveDefaultAddress(instruction *cpu65816.Instruction, value ast.Node) cpu65816.AddressingMode {
	if numberValue, ok := ast.NumberValue(value); ok {
		switch {
		case numberValue <= math.MaxUint8 && instruction.HasAddressing(cpu65816.DirectPageAddressing):
			return cpu65816.DirectPageAddressing
		case numberValue <= math.MaxUint16 && instruction.HasAddressing(cpu65816.AbsoluteAddressing):
			return cpu65816.AbsoluteAddressing
		case instruction.HasAddressing(cpu65816.AbsoluteLongAddressing):
			return cpu65816.AbsoluteLongAddressing
		}
	}
	hasDirect := instruction.HasAddressing(cpu65816.DirectPageAddressing)
	hasAbsolute := instruction.HasAddressing(cpu65816.AbsoluteAddressing)
	switch {
	case hasDirect && hasAbsolute:
		return AbsoluteDirectPageAddressing
	case hasAbsolute:
		return cpu65816.AbsoluteAddressing
	case hasDirect:
		return cpu65816.DirectPageAddressing
	case instruction.HasAddressing(cpu65816.AbsoluteLongAddressing):
		return cpu65816.AbsoluteLongAddressing
	default:
		return cpu65816.NoAddressing
	}
}

func resolveIndexedX(instruction *cpu65816.Instruction, operand Operand) cpu65816.AddressingMode {
	if operand.Size != AddressDefault {
		return sizedAddressing(
			operand.Size,
			cpu65816.DirectPageIndexedXAddressing,
			cpu65816.AbsoluteIndexedXAddressing,
			cpu65816.AbsoluteLongIndexedXAddressing,
		)
	}
	if instruction.HasAddressing(cpu65816.DirectPageIndexedXAddressing) &&
		instruction.HasAddressing(cpu65816.AbsoluteIndexedXAddressing) {

		return XAddressing
	}
	if numberValue, ok := ast.NumberValue(operand.Value); ok {
		switch {
		case numberValue <= math.MaxUint8 && instruction.HasAddressing(cpu65816.DirectPageIndexedXAddressing):
			return cpu65816.DirectPageIndexedXAddressing
		case numberValue <= math.MaxUint16 && instruction.HasAddressing(cpu65816.AbsoluteIndexedXAddressing):
			return cpu65816.AbsoluteIndexedXAddressing
		case instruction.HasAddressing(cpu65816.AbsoluteLongIndexedXAddressing):
			return cpu65816.AbsoluteLongIndexedXAddressing
		}
	}
	return firstSupported(
		instruction,
		cpu65816.AbsoluteIndexedXAddressing,
		cpu65816.DirectPageIndexedXAddressing,
		cpu65816.AbsoluteLongIndexedXAddressing,
	)
}

func resolveIndexedY(instruction *cpu65816.Instruction, operand Operand) cpu65816.AddressingMode {
	if operand.Size != AddressDefault {
		return sizedAddressing(
			operand.Size,
			cpu65816.DirectPageIndexedYAddressing,
			cpu65816.AbsoluteIndexedYAddressing,
			cpu65816.NoAddressing,
		)
	}
	if instruction.HasAddressing(cpu65816.DirectPageIndexedYAddressing) &&
		instruction.HasAddressing(cpu65816.AbsoluteIndexedYAddressing) {

		return YAddressing
	}
	if numberValue, ok := ast.NumberValue(operand.Value); ok {
		if numberValue <= math.MaxUint8 && instruction.HasAddressing(cpu65816.DirectPageIndexedYAddressing) {
			return cpu65816.DirectPageIndexedYAddressing
		}
		if numberValue <= math.MaxUint16 && instruction.HasAddressing(cpu65816.AbsoluteIndexedYAddressing) {
			return cpu65816.AbsoluteIndexedYAddressing
		}
	}
	return firstSupported(instruction, cpu65816.AbsoluteIndexedYAddressing, cpu65816.DirectPageIndexedYAddressing)
}

func resolveSizedPair(
	instruction *cpu65816.Instruction,
	operand Operand,
	directPage, absolute cpu65816.AddressingMode,
) cpu65816.AddressingMode {

	switch operand.Size {
	case AddressDirectPage:
		return directPage
	case AddressAbsolute:
		return absolute
	case AddressLong:
		return cpu65816.NoAddressing
	case AddressDefault:
		return firstSupported(instruction, directPage, absolute)
	default:
		return cpu65816.NoAddressing
	}
}

func sizedAddressing(
	size AddressSize,
	directPage, absolute, long cpu65816.AddressingMode,
) cpu65816.AddressingMode {

	switch size {
	case AddressDirectPage:
		return directPage
	case AddressAbsolute:
		return absolute
	case AddressLong:
		return long
	default:
		return cpu65816.NoAddressing
	}
}

func firstSupported(instruction *cpu65816.Instruction, addressings ...cpu65816.AddressingMode) cpu65816.AddressingMode {
	for _, addressing := range addressings {
		if instruction.HasAddressing(addressing) {
			return addressing
		}
	}
	return cpu65816.NoAddressing
}

func supportsAddressing(instruction *cpu65816.Instruction, addressing cpu65816.AddressingMode) bool {
	switch addressing {
	case AbsoluteDirectPageAddressing:
		return instruction.HasAddressing(cpu65816.AbsoluteAddressing) &&
			instruction.HasAddressing(cpu65816.DirectPageAddressing)
	case XAddressing:
		return instruction.HasAddressing(cpu65816.AbsoluteIndexedXAddressing) &&
			instruction.HasAddressing(cpu65816.DirectPageIndexedXAddressing)
	case YAddressing:
		return instruction.HasAddressing(cpu65816.AbsoluteIndexedYAddressing) &&
			instruction.HasAddressing(cpu65816.DirectPageIndexedYAddressing)
	default:
		return instruction.HasAddressing(addressing)
	}
}

func validateResolved(resolved ResolvedInstruction) error {
	if resolved.Instruction == nil {
		return fmt.Errorf("%w: missing instruction metadata", ErrInvalidInstruction)
	}
	if err := resolved.State.validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidInstruction, err)
	}
	addressing, err := resolveOperands(resolved.Instruction, resolved.Operands)
	if err != nil {
		return fmt.Errorf("%w: resolving operands for %s: %w", ErrInvalidInstruction, resolved.Instruction.Name, err)
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
			if _, err := number.Parse(modifier.Value); err != nil {
				return fmt.Errorf("%w: operand %d modifier %q: %w", ErrInvalidInstruction, index, modifier.Value, err)
			}
		}
	}
	return validateResolvedWidths(resolved)
}

func validateResolvedWidths(resolved ResolvedInstruction) error {
	if resolved.Addressing == cpu65816.ImmediateAddressing {
		width := immediateWidth(resolved.Instruction, resolved.State)
		if width == WidthUnknown {
			return fmt.Errorf("%w: %s immediate width is runtime-dependent", ErrInvalidInstruction, resolved.Instruction.Name)
		}
		return validateOperandMaximum(resolved.Operands, widthMaximum(width))
	}

	maximum := uint64(math.MaxUint32)
	switch resolved.Addressing {
	case cpu65816.DirectPageAddressing, cpu65816.DirectPageIndexedXAddressing,
		cpu65816.DirectPageIndexedYAddressing, cpu65816.DirectPageIndirectAddressing,
		cpu65816.DirectPageIndexedXIndirectAddressing, cpu65816.DirectPageIndirectIndexedYAddressing,
		cpu65816.DirectPageIndirectLongAddressing, cpu65816.DirectPageIndirectLongIndexedYAddressing,
		cpu65816.StackRelativeAddressing, cpu65816.StackRelativeIndirectIndexedYAddressing,
		cpu65816.BlockMoveAddressing:
		maximum = math.MaxUint8
	case cpu65816.AbsoluteAddressing, cpu65816.AbsoluteIndexedXAddressing,
		cpu65816.AbsoluteIndexedYAddressing, cpu65816.AbsoluteIndirectAddressing,
		cpu65816.AbsoluteIndexedXIndirectAddressing, cpu65816.AbsoluteIndirectLongAddressing:
		maximum = math.MaxUint16
	case cpu65816.AbsoluteLongAddressing, cpu65816.AbsoluteLongIndexedXAddressing,
		cpu65816.RelativeAddressing, cpu65816.RelativeLongAddressing,
		AbsoluteDirectPageAddressing, XAddressing, YAddressing:
		maximum = 0xffffff
	}
	return validateOperandMaximum(resolved.Operands, maximum)
}

func validateOperandMaximum(operands Operands, maximum uint64) error {
	for index, operand := range operands {
		if value, ok := ast.NumberValue(operand.Value); ok && value > maximum {
			return fmt.Errorf("operand %d value %d exceeds 0x%X", index, value, maximum)
		}
	}
	return nil
}

func widthMaximum(width Width) uint64 {
	if width == WidthWord {
		return math.MaxUint16
	}
	return math.MaxUint8
}
