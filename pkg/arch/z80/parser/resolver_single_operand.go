package parser

import (
	"fmt"

	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
)

func resolveSingleOperand(variants []*cpuz80.Instruction, operand rawOperand) (*ResolvedInstruction, error) {
	if result := resolveSingleRegisterOperand(variants, operand); result != nil {
		return result, nil
	}

	value, ok, err := operandValue(operand)
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

	// First pass: prefer variants without register opcodes.
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

	// Second pass: allow variants with register opcodes (e.g., SUB n has
	// RegisterOpcodes for register variants but also ImmediateAddressing).
	for _, variant := range variants {
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

	return nil, diagnoseSingleOperandMismatch(variants, operand)
}

func resolveSingleRegisterOperand(variants []*cpuz80.Instruction, operand rawOperand) *ResolvedInstruction {
	candidates := operandRegisterCandidates(operand)
	if len(candidates) == 0 {
		return nil
	}

	if result := matchRegisterOpcodeVariant(variants, operand, candidates); result != nil {
		return result
	}

	// Fallback: match parenthesized indirect register against Addressing map
	// for variants without RegisterOpcodes (e.g., JP (HL), INC (HL)).
	if operand.parenthesized && operand.displacement == nil {
		if result := resolveParenthesizedIndirect(variants); result != nil {
			return result
		}
	}

	// Fallback: for indexed operands (ix+d/iy+d), match via prefix in
	// RegisterOpcodes (e.g., SUB (IX+d) uses DD prefix with RegA as key).
	if operand.displacement != nil {
		return resolveIndexedSingleOperand(variants, operand)
	}

	return nil
}

func matchRegisterOpcodeVariant(variants []*cpuz80.Instruction, operand rawOperand, candidates []cpuz80.RegisterParam) *ResolvedInstruction {
	for _, variant := range variants {
		if len(variant.RegisterOpcodes) == 0 {
			continue
		}

		addressing, ok := selectRegisterAddressing(variant, operand.parenthesized)
		if !ok {
			continue
		}

		for _, candidate := range candidates {
			opcodeInfo, ok := variant.RegisterOpcodes[candidate]
			if !ok {
				continue
			}
			if !prefixMatchesIndexedBase(opcodeInfo.Prefix, operand) {
				continue
			}

			operandValues := make([]ast.Node, 0, 1)
			if operand.displacement != nil {
				operandValues = append(operandValues, operand.displacement)
			}

			return &ResolvedInstruction{
				Addressing:     addressing,
				Instruction:    variant,
				RegisterParams: []cpuz80.RegisterParam{candidate},
				OperandValues:  operandValues,
			}
		}
	}
	return nil
}

func resolveParenthesizedIndirect(variants []*cpuz80.Instruction) *ResolvedInstruction {
	for _, variant := range variants {
		if len(variant.RegisterOpcodes) != 0 {
			continue
		}
		if !variant.HasAddressing(cpuz80.RegisterIndirectAddressing) {
			continue
		}

		return &ResolvedInstruction{
			Addressing:  cpuz80.RegisterIndirectAddressing,
			Instruction: variant,
		}
	}
	return nil
}

func resolveIndexedSingleOperand(variants []*cpuz80.Instruction, operand rawOperand) *ResolvedInstruction {
	indexedRegister, ok := operandIndexedRegister(operand)
	if !ok {
		return nil
	}

	for _, variant := range variants {
		if !variant.HasAddressing(cpuz80.RegisterIndirectAddressing) {
			continue
		}

		for param, opcodeInfo := range variant.RegisterOpcodes {
			if !matchesIndexedRegisterPrefix(indexedRegister, opcodeInfo.Prefix) {
				continue
			}

			return &ResolvedInstruction{
				Addressing:     cpuz80.RegisterIndirectAddressing,
				Instruction:    variant,
				RegisterParams: []cpuz80.RegisterParam{param},
				OperandValues:  []ast.Node{operand.displacement},
			}
		}
	}

	return nil
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

func selectRegisterAddressing(variant *cpuz80.Instruction, parenthesized bool) (cpuz80.AddressingMode, bool) {
	if parenthesized && variant.HasAddressing(cpuz80.RegisterIndirectAddressing) {
		return cpuz80.RegisterIndirectAddressing, true
	}
	if parenthesized && variant.HasAddressing(cpuz80.PortAddressing) {
		return cpuz80.PortAddressing, true
	}
	if variant.HasAddressing(cpuz80.RegisterAddressing) {
		return cpuz80.RegisterAddressing, true
	}
	if variant.HasAddressing(cpuz80.ImpliedAddressing) {
		return cpuz80.ImpliedAddressing, true
	}

	if len(variant.Addressing) == 1 {
		for addressing := range variant.Addressing {
			return addressing, true
		}
	}

	return cpuz80.NoAddressing, false
}

func prefixMatchesIndexedBase(prefix byte, operand rawOperand) bool {
	indexedRegister, ok := operandIndexedRegister(operand)
	if !ok {
		return true
	}

	return matchesIndexedRegisterPrefix(indexedRegister, prefix)
}
