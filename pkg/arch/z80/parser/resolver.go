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

	return nil, errors.New("no implied-operand variant matched")
}

func resolveSingleOperand(variants []*cpuz80.Instruction, operand rawOperand) (*ResolvedInstruction, error) {
	if operand.token.Type == token.Identifier {
		if result := resolveSingleIdentifierOperand(variants, operand.token.Value); result != nil {
			return result, nil
		}
	}

	value, ok, err := parseValueOperand(operand.token)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%w: unsupported single operand type %s", errUnsupportedOperandPattern, operand.token.Type)
	}

	if numberValue, numberOK := value.(ast.Number); numberOK {
		if result := resolveSingleNumericRegisterOperand(variants, numberValue.Value); result != nil {
			return result, nil
		}
	}

	for _, variant := range variants {
		if len(variant.RegisterOpcodes) > 0 || len(variant.RegisterPairOpcodes) > 0 {
			continue
		}

		addressing, ok := selectValueAddressing(variant)
		if !ok {
			continue
		}

		return &ResolvedInstruction{
			Addressing:    addressing,
			Instruction:   variant,
			OperandValues: []ast.Node{value},
		}, nil
	}

	return nil, errors.New("no single-value variant matched")
}

func resolveSingleIdentifierOperand(variants []*cpuz80.Instruction, operand string) *ResolvedInstruction {
	candidates := registerCandidatesForIdentifier(operand)
	for _, variant := range variants {
		if !variant.HasAddressing(cpuz80.ImpliedAddressing) && !variant.HasAddressing(cpuz80.RegisterAddressing) {
			continue
		}
		for _, candidate := range candidates {
			if _, ok := variant.RegisterOpcodes[candidate]; !ok {
				continue
			}

			addressing := cpuz80.RegisterAddressing
			if variant.HasAddressing(cpuz80.ImpliedAddressing) {
				addressing = cpuz80.ImpliedAddressing
			}

			return &ResolvedInstruction{
				Addressing:     addressing,
				Instruction:    variant,
				RegisterParams: []cpuz80.RegisterParam{candidate},
			}
		}
	}

	return nil
}

func resolveTwoOperands(variants []*cpuz80.Instruction, operand1, operand2 rawOperand) (*ResolvedInstruction, error) {
	if result := resolveRegisterPairOperands(variants, operand1.token, operand2.token); result != nil {
		return result, nil
	}

	result, matched, err := resolveRegisterValueOperands(variants, operand1.token, operand2.token)
	if err != nil {
		return nil, err
	}
	if matched {
		return result, nil
	}

	result, matched, err = resolveValueRegisterOperands(variants, operand1.token, operand2.token)
	if err != nil {
		return nil, err
	}
	if matched {
		return result, nil
	}

	return nil, errors.New("no two-operand variant matched")
}

func resolveRegisterPairOperands(variants []*cpuz80.Instruction, token1, token2 token.Token) *ResolvedInstruction {
	if token1.Type != token.Identifier || token2.Type != token.Identifier {
		return nil
	}

	register1, ok := registerOnlyCandidate(token1.Value)
	if !ok {
		return nil
	}

	register2, ok := registerOnlyCandidate(token2.Value)
	if !ok {
		return nil
	}

	for _, variant := range variants {
		if _, ok := variant.RegisterPairOpcodes[[2]cpuz80.RegisterParam{register1, register2}]; !ok {
			continue
		}

		addressing := cpuz80.RegisterAddressing
		if variant.HasAddressing(cpuz80.RegisterIndirectAddressing) {
			addressing = cpuz80.RegisterIndirectAddressing
		}

		return &ResolvedInstruction{
			Addressing:     addressing,
			Instruction:    variant,
			RegisterParams: []cpuz80.RegisterParam{register1, register2},
		}
	}

	return nil
}

func resolveRegisterValueOperands(variants []*cpuz80.Instruction, token1, token2 token.Token) (*ResolvedInstruction, bool, error) {
	if token1.Type != token.Identifier {
		return nil, false, nil
	}

	value, ok, err := parseValueOperand(token2)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}

	candidates := registerCandidatesForIdentifier(token1.Value)
	for _, variant := range variants {
		addressing, ok := selectValueAddressing(variant)
		if !ok {
			continue
		}

		for _, candidate := range candidates {
			if _, ok := variant.RegisterOpcodes[candidate]; !ok {
				continue
			}

			return &ResolvedInstruction{
				Addressing:     addressing,
				Instruction:    variant,
				RegisterParams: []cpuz80.RegisterParam{candidate},
				OperandValues:  []ast.Node{value},
			}, true, nil
		}
	}

	return nil, false, nil
}

func resolveValueRegisterOperands(variants []*cpuz80.Instruction, token1, token2 token.Token) (*ResolvedInstruction, bool, error) {
	if token2.Type != token.Identifier {
		return nil, false, nil
	}

	value, ok, err := parseValueOperand(token1)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}

	candidates := registerCandidatesForIdentifier(token2.Value)
	for _, variant := range variants {
		addressing, ok := selectValueFirstAddressing(variant)
		if !ok {
			continue
		}

		for _, candidate := range candidates {
			if !supportsValueFirstRegister(variant, candidate) {
				continue
			}

			return &ResolvedInstruction{
				Addressing:     addressing,
				Instruction:    variant,
				RegisterParams: []cpuz80.RegisterParam{candidate},
				OperandValues:  []ast.Node{value},
			}, true, nil
		}
	}

	return nil, false, nil
}

func resolveSingleNumericRegisterOperand(variants []*cpuz80.Instruction, value uint64) *ResolvedInstruction {
	candidates := registerCandidatesForNumber(value)
	if len(candidates) == 0 {
		return nil
	}

	for _, variant := range variants {
		if len(variant.RegisterOpcodes) == 0 {
			continue
		}

		for _, candidate := range candidates {
			if _, ok := variant.RegisterOpcodes[candidate]; !ok {
				continue
			}

			addressing := cpuz80.NoAddressing
			if len(variant.Addressing) == 1 {
				for mode := range variant.Addressing {
					addressing = mode
				}
			}
			if addressing == cpuz80.NoAddressing && variant.HasAddressing(cpuz80.ImpliedAddressing) {
				addressing = cpuz80.ImpliedAddressing
			}
			if addressing == cpuz80.NoAddressing {
				addressing = cpuz80.RegisterAddressing
			}

			return &ResolvedInstruction{
				Addressing:     addressing,
				Instruction:    variant,
				RegisterParams: []cpuz80.RegisterParam{candidate},
			}
		}
	}

	return nil
}

func selectValueFirstAddressing(variant *cpuz80.Instruction) (cpuz80.AddressingMode, bool) {
	switch {
	case variant.HasAddressing(cpuz80.BitAddressing):
		return cpuz80.BitAddressing, true
	case variant.HasAddressing(cpuz80.RegisterAddressing):
		return cpuz80.RegisterAddressing, true
	default:
		return cpuz80.NoAddressing, false
	}
}

func supportsValueFirstRegister(variant *cpuz80.Instruction, register cpuz80.RegisterParam) bool {
	if _, ok := variant.RegisterOpcodes[register]; ok {
		return true
	}

	if isBitOperation(variant) {
		return isBitTargetRegister(register)
	}

	return false
}

func isBitOperation(variant *cpuz80.Instruction) bool {
	return variant == cpuz80.CBBit || variant == cpuz80.CBRes || variant == cpuz80.CBSet
}

func isBitTargetRegister(register cpuz80.RegisterParam) bool {
	switch register {
	case cpuz80.RegB, cpuz80.RegC, cpuz80.RegD, cpuz80.RegE, cpuz80.RegH, cpuz80.RegL, cpuz80.RegHLIndirect, cpuz80.RegA:
		return true
	default:
		return false
	}
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
