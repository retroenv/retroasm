package parser

import (
	"errors"
	"fmt"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/number"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
)

var errUnsupportedOperandPattern = errors.New("unsupported operand pattern")

// ResolvedInstruction contains the selected Z80 instruction variant and parsed operand data.
type ResolvedInstruction struct {
	Addressing     cpuz80.AddressingMode
	Instruction    *cpuz80.Instruction
	RegisterParams []cpuz80.RegisterParam
	OperandValues  []ast.Node
}

type rawOperand struct {
	token token.Token

	value        ast.Node
	displacement ast.Node

	parenthesized  bool
	registerParams []cpuz80.RegisterParam
}

func resolveInstruction(variants []*cpuz80.Instruction, operands []rawOperand) (*ResolvedInstruction, error) {
	switch len(operands) {
	case 0:
		return resolveNoOperand(variants)
	case 1:
		return resolveSingleOperand(variants, operands[0])
	case 2:
		return resolveTwoOperands(variants, operands[0], operands[1])
	default:
		return nil, fmt.Errorf("%w: expected at most 2 operands, got %d", errUnsupportedOperandPattern, len(operands))
	}
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
