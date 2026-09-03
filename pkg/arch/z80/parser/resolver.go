package parser

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/number"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
)

var errUnsupportedOperandPattern = errors.New("unsupported operand pattern")

var (
	// ErrMissingInstruction indicates incomplete resolved instruction metadata.
	ErrMissingInstruction = errors.New("resolved instruction details are missing")
	// ErrOpcodeNotFound indicates that resolved operands do not select an encoded variant.
	ErrOpcodeNotFound = errors.New("opcode mapping not found")
)

// ResolvedInstruction contains the selected Z80 instruction variant and parsed operand data.
type ResolvedInstruction struct {
	Addressing     cpuz80.AddressingMode
	Instruction    *cpuz80.Instruction
	RegisterParams []cpuz80.RegisterParam
	OperandValues  []ast.Node
	Operands       []Operand
}

// CopyInstructionArgument returns a deep copy suitable for AST duplication.
func (resolved ResolvedInstruction) CopyInstructionArgument() any {
	resolved.RegisterParams = slices.Clone(resolved.RegisterParams)
	resolved.OperandValues = ast.CopyNodes(resolved.OperandValues)
	resolved.Operands = copyOperands(resolved.Operands)
	return resolved
}

// InstructionFormKey returns the selected addressing, register, and operand shapes.
func (resolved ResolvedInstruction) InstructionFormKey() string {
	registers := make([]string, len(resolved.RegisterParams))
	for index, register := range resolved.RegisterParams {
		registers[index] = fmt.Sprintf("%d", register)
	}
	operands := make([]string, len(resolved.Operands))
	for index, operand := range resolved.Operands {
		operands[index] = fmt.Sprintf(
			"%d/%d/%s",
			operand.Kind,
			operand.Register,
			ast.InstructionArgumentForm(operand.Value),
		)
	}
	return fmt.Sprintf(
		"addressing=%d;registers=%s;operands=%s",
		resolved.Addressing,
		strings.Join(registers, ","),
		strings.Join(operands, ","),
	)
}

// InstructionReferences returns copied symbol-bearing values for stream validation.
func (resolved ResolvedInstruction) InstructionReferences() []ast.InstructionReference {
	references := make([]ast.InstructionReference, 0, len(resolved.OperandValues))

	for _, value := range resolved.OperandValues {
		if !operandReferencesSymbol(value) {
			continue
		}
		references = append(references, ast.InstructionReference{
			Value:         value.Copy(),
			ReferenceType: ast.FullAddress,
		})
	}
	return references
}

// OpcodeInfo returns the selected encoded form and effective addressing mode.
func (resolved ResolvedInstruction) OpcodeInfo() (cpuz80.OpcodeInfo, cpuz80.AddressingMode, error) {
	if resolved.Instruction == nil {
		return cpuz80.OpcodeInfo{}, cpuz80.NoAddressing, ErrMissingInstruction
	}

	if len(resolved.RegisterParams) > 0 {
		switch len(resolved.RegisterParams) {
		case 1:
			if info, ok := resolved.Instruction.RegisterOpcodes[resolved.RegisterParams[0]]; ok {
				return info, resolved.effectiveAddressing(), nil
			}
		case 2:
			key := [2]cpuz80.RegisterParam{resolved.RegisterParams[0], resolved.RegisterParams[1]}
			if info, ok := resolved.Instruction.RegisterPairOpcodes[key]; ok {
				return info, resolved.effectiveAddressing(), nil
			}
		}
	}

	if resolved.Addressing != cpuz80.NoAddressing {
		if info, ok := resolved.Instruction.Addressing[resolved.Addressing]; ok {
			return info, resolved.Addressing, nil
		}
		return cpuz80.OpcodeInfo{}, cpuz80.NoAddressing, ErrOpcodeNotFound
	}
	if len(resolved.Instruction.Addressing) == 1 {
		for addressing, info := range resolved.Instruction.Addressing {
			return info, addressing, nil
		}
	}
	return cpuz80.OpcodeInfo{}, cpuz80.NoAddressing, ErrOpcodeNotFound
}

func (resolved ResolvedInstruction) effectiveAddressing() cpuz80.AddressingMode {
	if resolved.Addressing != cpuz80.NoAddressing {
		return resolved.Addressing
	}
	if len(resolved.Instruction.Addressing) == 1 {
		for addressing := range resolved.Instruction.Addressing {
			return addressing
		}
	}
	return cpuz80.NoAddressing
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

type rawOperand struct {
	token token.Token

	value        ast.Node
	displacement ast.Node

	parenthesized  bool
	registerParams []cpuz80.RegisterParam
}

func resolveInstruction(variants []*cpuz80.Instruction, operands []rawOperand) (*ResolvedInstruction, error) {
	var resolved *ResolvedInstruction
	var err error
	switch len(operands) {
	case 0:
		resolved, err = resolveNoOperand(variants)
	case 1:
		resolved, err = resolveSingleOperand(variants, operands[0])
	case 2:
		resolved, err = resolveTwoOperands(variants, operands[0], operands[1])
	default:
		return nil, fmt.Errorf("%w: expected at most 2 operands, got %d", errUnsupportedOperandPattern, len(operands))
	}
	if err != nil {
		return nil, err
	}

	resolved.Operands, err = operandsFromRaw(operands, resolved.OperandValues)
	if err != nil {
		return nil, fmt.Errorf("retaining resolved operands: %w", err)
	}
	return resolved, nil
}

func resolveNoOperand(variants []*cpuz80.Instruction) (*ResolvedInstruction, error) {
	// First pass: prefer variants without register opcodes.
	for _, variant := range variants {
		if !variant.HasAddressing(cpuz80.ImpliedAddressing) {
			continue
		}
		if len(variant.RegisterOpcodes) > 0 || len(variant.RegisterPairOpcodes) > 0 {
			continue
		}

		return &ResolvedInstruction{
			Addressing:  cpuz80.ImpliedAddressing,
			Instruction: variant,
		}, nil
	}

	// Second pass: allow variants with register opcodes (e.g., NEG, RETN have
	// undocumented register variants but are used without operands).
	for _, variant := range variants {
		if !variant.HasAddressing(cpuz80.ImpliedAddressing) {
			continue
		}

		return &ResolvedInstruction{
			Addressing:  cpuz80.ImpliedAddressing,
			Instruction: variant,
		}, nil
	}

	return nil, noMatchDiagnostic("no implied-operand variant matched", variants)
}

func selectValueAddressing(variant *cpuz80.Instruction) (cpuz80.AddressingMode, bool) {
	switch {
	case variant.HasAddressing(cpuz80.RelativeAddressing):
		return cpuz80.RelativeAddressing, true
	case variant.HasAddressing(cpuz80.ExtendedAddressing):
		return cpuz80.ExtendedAddressing, true
	case variant.HasAddressing(cpuz80.ImmediateAddressing):
		return cpuz80.ImmediateAddressing, true
	default:
		return cpuz80.NoAddressing, false
	}
}

func matchesIndexedRegisterPrefix(indexedRegister cpuz80.RegisterParam, prefix byte) bool {
	switch indexedRegister {
	case cpuz80.RegIXIndirect:
		return prefix == cpuz80.PrefixDD
	case cpuz80.RegIYIndirect:
		return prefix == cpuz80.PrefixFD
	default:
		return false
	}
}

func operandValue(operand rawOperand) (ast.Node, bool, error) {
	if operand.value != nil {
		return operand.value, true, nil
	}

	return parseValueOperand(operand.token)
}

func operandRegisterCandidates(operand rawOperand) []cpuz80.RegisterParam {
	if len(operand.registerParams) > 0 {
		return operand.registerParams
	}

	if operand.token.Type == token.Identifier {
		return registerCandidatesForIdentifier(operand.token.Value)
	}

	return nil
}

func operandRegisterOnlyCandidates(operand rawOperand) []cpuz80.RegisterParam {
	if len(operand.registerParams) > 0 {
		return operand.registerParams
	}

	if operand.token.Type != token.Identifier {
		return nil
	}

	registerParam, ok := registerOnlyCandidate(operand.token.Value)
	if !ok {
		return nil
	}
	return []cpuz80.RegisterParam{registerParam}
}

func operandIndexedRegister(operand rawOperand) (cpuz80.RegisterParam, bool) {
	if operand.displacement == nil {
		return cpuz80.RegNone, false
	}

	for _, candidate := range operandRegisterCandidates(operand) {
		switch candidate {
		case cpuz80.RegIXIndirect, cpuz80.RegIYIndirect:
			return candidate, true
		}
	}

	return cpuz80.RegNone, false
}

func parseValueOperand(tok token.Token) (ast.Node, bool, error) {
	switch tok.Type {
	case token.Number:
		value, err := number.Parse(tok.Value)
		if err != nil {
			return nil, false, fmt.Errorf("parsing number '%s': %w", tok.Value, err)
		}
		return ast.NewNumber(value), true, nil
	case token.Identifier:
		return ast.NewLabel(tok.Value), true, nil
	default:
		return nil, false, nil
	}
}
