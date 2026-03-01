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

	return nil, noMatchDiagnostic("no implied-operand variant matched", variants)
}

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

	return nil, diagnoseSingleOperandMismatch(variants, operand)
}

func resolveSingleRegisterOperand(variants []*cpuz80.Instruction, operand rawOperand) *ResolvedInstruction {
	candidates := operandRegisterCandidates(operand)
	if len(candidates) == 0 {
		return nil
	}

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

func resolveTwoOperands(variants []*cpuz80.Instruction, operand1, operand2 rawOperand) (*ResolvedInstruction, error) {
	if result := resolveRegisterPairOperands(variants, operand1, operand2); result != nil {
		return result, nil
	}

	result, matched, err := resolveExtendedRegisterMemoryOperands(variants, operand1, operand2)
	if err != nil {
		return nil, err
	}
	if matched {
		return result, nil
	}

	result, matched = resolvePortRegisterOperands(variants, operand1, operand2)
	if matched {
		return result, nil
	}

	result, matched, err = resolvePortImmediateOperands(variants, operand1, operand2)
	if err != nil {
		return nil, err
	}
	if matched {
		return result, nil
	}

	result, matched = resolveRegisterIndexedOperands(variants, operand1, operand2)
	if matched {
		return result, nil
	}

	result, matched = resolveIndexedRegisterOperands(variants, operand1, operand2)
	if matched {
		return result, nil
	}

	result, matched, err = resolveRegisterValueOperands(variants, operand1, operand2)
	if err != nil {
		return nil, err
	}
	if matched {
		return result, nil
	}

	result, matched, err = resolveValueRegisterOperands(variants, operand1, operand2)
	if err != nil {
		return nil, err
	}
	if matched {
		return result, nil
	}

	return nil, diagnoseTwoOperandMismatch(variants, operand1, operand2)
}

func resolveRegisterPairOperands(variants []*cpuz80.Instruction, operand1, operand2 rawOperand) *ResolvedInstruction {
	if operand1.displacement != nil || operand2.displacement != nil {
		return nil
	}

	candidates1 := operandRegisterOnlyCandidates(operand1)
	if len(candidates1) == 0 {
		return nil
	}

	candidates2 := operandRegisterOnlyCandidates(operand2)
	if len(candidates2) == 0 {
		return nil
	}

	for _, variant := range variants {
		for _, register1 := range candidates1 {
			for _, register2 := range candidates2 {
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
		}
	}

	return nil
}

func resolveRegisterValueOperands(variants []*cpuz80.Instruction, operand1, operand2 rawOperand) (*ResolvedInstruction, bool, error) {
	candidates := operandRegisterCandidates(operand1)
	if len(candidates) == 0 {
		return nil, false, nil
	}

	value, ok, err := operandValue(operand2)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}

	for _, variant := range variants {
		addressing, ok := selectValueAddressing(variant)
		if !ok {
			continue
		}
		if operand2.parenthesized && addressing == cpuz80.ImmediateAddressing {
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

func resolveExtendedRegisterMemoryOperands(variants []*cpuz80.Instruction, operand1, operand2 rawOperand) (*ResolvedInstruction, bool, error) {
	if operand1.displacement != nil || operand2.displacement != nil {
		return nil, false, nil
	}

	result, matched, err := resolveRegisterFromExtendedMemory(variants, operand1, operand2)
	if matched || err != nil {
		return result, matched, err
	}

	return resolveExtendedMemoryFromRegister(variants, operand1, operand2)
}

func resolvePortImmediateOperands(variants []*cpuz80.Instruction, operand1, operand2 rawOperand) (*ResolvedInstruction, bool, error) {
	if operand1.displacement != nil || operand2.displacement != nil {
		return nil, false, nil
	}

	result, matched, err := resolvePortImmediateValueRegister(variants, operand1, operand2)
	if matched || err != nil {
		return result, matched, err
	}

	return resolvePortImmediateRegisterValue(variants, operand1, operand2)
}

func resolveRegisterFromExtendedMemory(variants []*cpuz80.Instruction, registerOperand, valueOperand rawOperand) (*ResolvedInstruction, bool, error) {
	if !valueOperand.parenthesized {
		return nil, false, nil
	}

	candidates := operandRegisterCandidates(registerOperand)
	if len(candidates) == 0 {
		return nil, false, nil
	}

	value, ok, err := operandValue(valueOperand)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}

	for _, variant := range variants {
		if !variant.HasAddressing(cpuz80.ExtendedAddressing) {
			continue
		}

		for _, candidate := range candidates {
			for _, resolvedParam := range extendedRegisterParamCandidates(candidate, true) {
				opcodeInfo, ok := variant.RegisterOpcodes[resolvedParam]
				if !ok {
					continue
				}
				if !matchesExtendedLoadDirection(variant, opcodeInfo.Opcode, true) {
					continue
				}

				return &ResolvedInstruction{
					Addressing:     cpuz80.ExtendedAddressing,
					Instruction:    variant,
					RegisterParams: []cpuz80.RegisterParam{resolvedParam},
					OperandValues:  []ast.Node{value},
				}, true, nil
			}
		}
	}

	return nil, false, nil
}

func resolveExtendedMemoryFromRegister(variants []*cpuz80.Instruction, valueOperand, registerOperand rawOperand) (*ResolvedInstruction, bool, error) {
	if !valueOperand.parenthesized {
		return nil, false, nil
	}

	candidates := operandRegisterCandidates(registerOperand)
	if len(candidates) == 0 {
		return nil, false, nil
	}

	value, ok, err := operandValue(valueOperand)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}

	for _, variant := range variants {
		if !variant.HasAddressing(cpuz80.ExtendedAddressing) {
			continue
		}

		for _, candidate := range candidates {
			for _, resolvedParam := range extendedRegisterParamCandidates(candidate, false) {
				opcodeInfo, ok := variant.RegisterOpcodes[resolvedParam]
				if !ok {
					continue
				}
				if !matchesExtendedLoadDirection(variant, opcodeInfo.Opcode, false) {
					continue
				}

				return &ResolvedInstruction{
					Addressing:     cpuz80.ExtendedAddressing,
					Instruction:    variant,
					RegisterParams: []cpuz80.RegisterParam{resolvedParam},
					OperandValues:  []ast.Node{value},
				}, true, nil
			}

			if candidate != cpuz80.RegHL || len(variant.RegisterOpcodes) != 0 {
				continue
			}

			return &ResolvedInstruction{
				Addressing:    cpuz80.ExtendedAddressing,
				Instruction:   variant,
				OperandValues: []ast.Node{value},
			}, true, nil
		}
	}

	return nil, false, nil
}

func resolvePortImmediateValueRegister(variants []*cpuz80.Instruction, valueOperand, registerOperand rawOperand) (*ResolvedInstruction, bool, error) {
	if !valueOperand.parenthesized {
		return nil, false, nil
	}

	registerCandidates := operandRegisterCandidates(registerOperand)
	if !containsRegisterParam(registerCandidates, cpuz80.RegA) {
		return nil, false, nil
	}

	value, ok, err := operandValue(valueOperand)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}

	for _, variant := range variants {
		if !variant.HasAddressing(cpuz80.PortAddressing) || len(variant.RegisterOpcodes) != 0 {
			continue
		}

		return &ResolvedInstruction{
			Addressing:    cpuz80.PortAddressing,
			Instruction:   variant,
			OperandValues: []ast.Node{value},
		}, true, nil
	}

	return nil, false, nil
}

func resolvePortImmediateRegisterValue(variants []*cpuz80.Instruction, registerOperand, valueOperand rawOperand) (*ResolvedInstruction, bool, error) {
	if !valueOperand.parenthesized {
		return nil, false, nil
	}

	registerCandidates := operandRegisterCandidates(registerOperand)
	if !containsRegisterParam(registerCandidates, cpuz80.RegA) {
		return nil, false, nil
	}

	value, ok, err := operandValue(valueOperand)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}

	for _, variant := range variants {
		if !variant.HasAddressing(cpuz80.PortAddressing) || len(variant.RegisterOpcodes) != 0 {
			continue
		}

		return &ResolvedInstruction{
			Addressing:    cpuz80.PortAddressing,
			Instruction:   variant,
			OperandValues: []ast.Node{value},
		}, true, nil
	}

	return nil, false, nil
}

func resolvePortRegisterOperands(variants []*cpuz80.Instruction, operand1, operand2 rawOperand) (*ResolvedInstruction, bool) {
	if operand1.displacement != nil || operand2.displacement != nil {
		return nil, false
	}

	if isPortCOperand(operand2) {
		candidates := operandRegisterCandidates(operand1)
		for _, variant := range variants {
			if !variant.HasAddressing(cpuz80.PortAddressing) || len(variant.RegisterOpcodes) == 0 {
				continue
			}
			for _, candidate := range candidates {
				if _, ok := variant.RegisterOpcodes[candidate]; !ok {
					continue
				}
				return &ResolvedInstruction{
					Addressing:     cpuz80.PortAddressing,
					Instruction:    variant,
					RegisterParams: []cpuz80.RegisterParam{candidate},
				}, true
			}
		}
	}

	if isPortCOperand(operand1) {
		candidates := operandRegisterCandidates(operand2)
		for _, variant := range variants {
			if !variant.HasAddressing(cpuz80.PortAddressing) || len(variant.RegisterOpcodes) == 0 {
				continue
			}
			for _, candidate := range candidates {
				if _, ok := variant.RegisterOpcodes[candidate]; !ok {
					continue
				}
				return &ResolvedInstruction{
					Addressing:     cpuz80.PortAddressing,
					Instruction:    variant,
					RegisterParams: []cpuz80.RegisterParam{candidate},
				}, true
			}
		}
	}

	return nil, false
}

func resolveRegisterIndexedOperands(variants []*cpuz80.Instruction, operand1, operand2 rawOperand) (*ResolvedInstruction, bool) {
	if operand2.displacement == nil {
		return nil, false
	}

	indexedRegister, ok := operandIndexedRegister(operand2)
	if !ok {
		return nil, false
	}

	candidates := operandRegisterCandidates(operand1)
	if len(candidates) == 0 {
		return nil, false
	}

	for _, variant := range variants {
		if !variant.HasAddressing(cpuz80.RegisterIndirectAddressing) {
			continue
		}

		for _, candidate := range candidates {
			opcodeInfo, ok := variant.RegisterOpcodes[candidate]
			if !ok {
				continue
			}
			if !matchesIndexedRegisterPrefix(indexedRegister, opcodeInfo.Prefix) {
				continue
			}
			if !matchesIndexedLoadDirection(variant, opcodeInfo.Opcode, true) {
				continue
			}

			return &ResolvedInstruction{
				Addressing:     cpuz80.RegisterIndirectAddressing,
				Instruction:    variant,
				RegisterParams: []cpuz80.RegisterParam{candidate},
				OperandValues:  []ast.Node{operand2.displacement},
			}, true
		}
	}

	return nil, false
}

func resolveIndexedRegisterOperands(variants []*cpuz80.Instruction, operand1, operand2 rawOperand) (*ResolvedInstruction, bool) {
	if operand1.displacement == nil {
		return nil, false
	}

	indexedRegister, ok := operandIndexedRegister(operand1)
	if !ok {
		return nil, false
	}

	candidates := operandRegisterCandidates(operand2)
	if len(candidates) == 0 {
		return nil, false
	}

	for _, variant := range variants {
		if !variant.HasAddressing(cpuz80.RegisterIndirectAddressing) {
			continue
		}

		for _, candidate := range candidates {
			opcodeInfo, ok := variant.RegisterOpcodes[candidate]
			if !ok {
				continue
			}
			if !matchesIndexedRegisterPrefix(indexedRegister, opcodeInfo.Prefix) {
				continue
			}
			if !matchesIndexedLoadDirection(variant, opcodeInfo.Opcode, false) {
				continue
			}

			return &ResolvedInstruction{
				Addressing:     cpuz80.RegisterIndirectAddressing,
				Instruction:    variant,
				RegisterParams: []cpuz80.RegisterParam{candidate},
				OperandValues:  []ast.Node{operand1.displacement},
			}, true
		}
	}

	return nil, false
}

func resolveValueRegisterOperands(variants []*cpuz80.Instruction, operand1, operand2 rawOperand) (*ResolvedInstruction, bool, error) {
	value, ok, err := operandValue(operand1)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}

	candidates := operandRegisterCandidates(operand2)
	if len(candidates) == 0 {
		return nil, false, nil
	}

	for _, variant := range variants {
		addressing, ok := selectValueFirstAddressing(variant)
		if !ok {
			continue
		}

		for _, candidate := range candidates {
			resolvedRegister, ok := resolveValueFirstRegister(variant, candidate)
			if !ok {
				continue
			}

			operandValues := []ast.Node{value}
			if operand2.displacement != nil {
				operandValues = append(operandValues, operand2.displacement)
			}

			return &ResolvedInstruction{
				Addressing:     addressing,
				Instruction:    variant,
				RegisterParams: []cpuz80.RegisterParam{resolvedRegister},
				OperandValues:  operandValues,
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

func resolveValueFirstRegister(variant *cpuz80.Instruction, register cpuz80.RegisterParam) (cpuz80.RegisterParam, bool) {
	if _, ok := variant.RegisterOpcodes[register]; ok {
		return register, true
	}

	if isBitOperation(variant) {
		if isBitTargetRegister(register) {
			return register, true
		}

		switch {
		case register == cpuz80.RegIXIndirect && isDdcbBitOperation(variant):
			return cpuz80.RegHLIndirect, true
		case register == cpuz80.RegIYIndirect && isFdcbBitOperation(variant):
			return cpuz80.RegHLIndirect, true
		}
	}

	return cpuz80.RegNone, false
}

func isBitOperation(variant *cpuz80.Instruction) bool {
	return variant == cpuz80.CBBit || variant == cpuz80.CBRes || variant == cpuz80.CBSet ||
		isDdcbBitOperation(variant) || isFdcbBitOperation(variant)
}

func isDdcbBitOperation(variant *cpuz80.Instruction) bool {
	return variant == cpuz80.DdcbBit || variant == cpuz80.DdcbRes || variant == cpuz80.DdcbSet
}

func isFdcbBitOperation(variant *cpuz80.Instruction) bool {
	return variant == cpuz80.FdcbBit || variant == cpuz80.FdcbRes || variant == cpuz80.FdcbSet
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

func matchesIndexedLoadDirection(variant *cpuz80.Instruction, opcode byte, registerFirst bool) bool {
	if variant.Name != cpuz80.LdName {
		return true
	}

	if registerFirst {
		return opcode&0x07 == 0x06
	}

	return opcode&0xF8 == 0x70
}

func matchesExtendedLoadDirection(variant *cpuz80.Instruction, opcode byte, registerFirst bool) bool {
	if variant.Name != cpuz80.LdName {
		return true
	}

	if registerFirst {
		return opcode&0x0F == 0x0B || opcode == 0x2A || opcode == 0x3A
	}

	return opcode&0x0F == 0x03 || opcode == 0x22 || opcode == 0x32
}

func extendedRegisterParamCandidates(register cpuz80.RegisterParam, registerFirst bool) []cpuz80.RegisterParam {
	switch register {
	case cpuz80.RegA:
		if registerFirst {
			return []cpuz80.RegisterParam{cpuz80.RegLoadExtA, cpuz80.RegA}
		}
		return []cpuz80.RegisterParam{cpuz80.RegStoreExtA, cpuz80.RegA}
	case cpuz80.RegHL:
		if registerFirst {
			return []cpuz80.RegisterParam{cpuz80.RegLoadExtHL, cpuz80.RegHL}
		}
		return []cpuz80.RegisterParam{cpuz80.RegHL}
	default:
		return []cpuz80.RegisterParam{register}
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

func isPortCOperand(operand rawOperand) bool {
	if !operand.parenthesized || operand.value != nil || operand.displacement != nil {
		return false
	}

	return containsRegisterParam(operandRegisterCandidates(operand), cpuz80.RegC)
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

func diagnoseSingleOperandMismatch(variants []*cpuz80.Instruction, operand rawOperand) error {
	if isConditionRegisterCAmbiguity(variants, operand, rawOperand{}) {
		return fmt.Errorf(
			"ambiguous operand 'c': it may be carry condition or register c; use condition forms like 'jp c,label' or register forms like '(c)'; expected addressing families: %s",
			expectedAddressingFamilies(variants),
		)
	}

	if isImmediateVsAddressedMismatch(variants, operand, rawOperand{}) {
		return fmt.Errorf(
			"immediate vs addressed operand mismatch: use n/nn for immediate operands and parenthesized forms like (nn), (hl), (c), (ix+d), or (iy+d) for addressed operands; expected addressing families: %s",
			expectedAddressingFamilies(variants),
		)
	}

	return noMatchDiagnostic("no single-operand variant matched", variants)
}

func diagnoseTwoOperandMismatch(variants []*cpuz80.Instruction, operand1, operand2 rawOperand) error {
	if isConditionRegisterCAmbiguity(variants, operand1, operand2) {
		return fmt.Errorf(
			"ambiguous operand 'c': it may be carry condition or register c; use condition forms like 'jp c,label' or register forms like '(c)'; expected addressing families: %s",
			expectedAddressingFamilies(variants),
		)
	}

	if isIndexedLoadDirectionMismatch(variants, operand1, operand2) {
		return fmt.Errorf(
			"indexed load direction mismatch: use 'ld r,(ix+d|iy+d)' for indexed reads and 'ld (ix+d|iy+d),r' for indexed stores; expected addressing families: %s",
			expectedAddressingFamilies(variants),
		)
	}

	if isImmediateVsAddressedMismatch(variants, operand1, operand2) {
		return fmt.Errorf(
			"immediate vs addressed operand mismatch: use n/nn for immediate operands and parenthesized forms like (nn), (hl), (c), (ix+d), or (iy+d) for addressed operands; expected addressing families: %s",
			expectedAddressingFamilies(variants),
		)
	}

	return noMatchDiagnostic("no two-operand variant matched", variants)
}

func noMatchDiagnostic(message string, variants []*cpuz80.Instruction) error {
	return fmt.Errorf("%s; expected addressing families: %s", message, expectedAddressingFamilies(variants))
}

func expectedAddressingFamilies(variants []*cpuz80.Instruction) string {
	families := make([]string, 0, 8)
	for family := range addressingFamilies(variants) {
		families = append(families, family)
	}
	if len(families) == 0 {
		return "unknown"
	}

	slices.Sort(families)
	return strings.Join(families, ", ")
}

func addressingFamilies(variants []*cpuz80.Instruction) map[string]struct{} {
	families := make(map[string]struct{}, 8)

	for _, variant := range variants {
		for addressing := range variant.Addressing {
			families[addressingFamilyName(addressing)] = struct{}{}
		}
		if len(variant.RegisterOpcodes) > 0 || len(variant.RegisterPairOpcodes) > 0 {
			families["register"] = struct{}{}
		}
	}

	return families
}

func addressingFamilyName(addressing cpuz80.AddressingMode) string {
	switch addressing {
	case cpuz80.ImpliedAddressing:
		return "implied"
	case cpuz80.RegisterAddressing:
		return "register"
	case cpuz80.ImmediateAddressing:
		return "immediate"
	case cpuz80.ExtendedAddressing:
		return "extended"
	case cpuz80.RegisterIndirectAddressing:
		return "register-indirect"
	case cpuz80.RelativeAddressing:
		return "relative"
	case cpuz80.BitAddressing:
		return "bit"
	case cpuz80.PortAddressing:
		return "port"
	default:
		return fmt.Sprintf("mode-%d", addressing)
	}
}

func isConditionRegisterCAmbiguity(variants []*cpuz80.Instruction, operand1, operand2 rawOperand) bool {
	if !isIdentifierOperand(operand1, "c") {
		return false
	}
	if !variantsSupportConditionOperand(variants) {
		return false
	}

	if operand2.token.Type == token.Identifier && !operand2.parenthesized {
		return true
	}

	return operand2.parenthesized || operand2.displacement != nil || operand2.value != nil
}

func isImmediateVsAddressedMismatch(variants []*cpuz80.Instruction, operand1, operand2 rawOperand) bool {
	if !variantsHaveAddressing(variants, cpuz80.ImmediateAddressing, cpuz80.ExtendedAddressing, cpuz80.PortAddressing, cpuz80.RegisterIndirectAddressing) {
		return false
	}

	plainValue := hasPlainValueOperand(operand1) || hasPlainValueOperand(operand2)
	parenthesizedValue := hasParenthesizedValueOperand(operand1) || hasParenthesizedValueOperand(operand2)

	return plainValue || parenthesizedValue
}

func isIndexedLoadDirectionMismatch(variants []*cpuz80.Instruction, operand1, operand2 rawOperand) bool {
	if operand1.displacement == nil && operand2.displacement == nil {
		return false
	}
	if !variantsContainMnemonic(variants, cpuz80.LdName) {
		return false
	}
	if !variantsHaveIndexedPrefix(variants) {
		return false
	}

	if operand1.displacement != nil && len(operandRegisterCandidates(operand2)) > 0 {
		return true
	}

	return operand2.displacement != nil && len(operandRegisterCandidates(operand1)) > 0
}

func variantsContainMnemonic(variants []*cpuz80.Instruction, mnemonic string) bool {
	for _, variant := range variants {
		if variant.Name == mnemonic {
			return true
		}
	}

	return false
}

func variantsHaveAddressing(variants []*cpuz80.Instruction, addressings ...cpuz80.AddressingMode) bool {
	for _, variant := range variants {
		for _, addressing := range addressings {
			if variant.HasAddressing(addressing) {
				return true
			}
		}
	}

	return false
}

func variantsHaveIndexedPrefix(variants []*cpuz80.Instruction) bool {
	for _, variant := range variants {
		for _, info := range variant.Addressing {
			if info.Prefix == cpuz80.PrefixDD || info.Prefix == cpuz80.PrefixFD {
				return true
			}
		}

		for _, info := range variant.RegisterOpcodes {
			if info.Prefix == cpuz80.PrefixDD || info.Prefix == cpuz80.PrefixFD {
				return true
			}
		}
	}

	return false
}

func variantsSupportConditionOperand(variants []*cpuz80.Instruction) bool {
	for _, variant := range variants {
		for registerParam := range variant.RegisterOpcodes {
			if isConditionRegister(registerParam) {
				return true
			}
		}
	}

	return false
}

func isConditionRegister(registerParam cpuz80.RegisterParam) bool {
	switch registerParam {
	case cpuz80.RegCondNZ,
		cpuz80.RegCondZ,
		cpuz80.RegCondNC,
		cpuz80.RegCondC,
		cpuz80.RegCondPO,
		cpuz80.RegCondPE,
		cpuz80.RegCondP,
		cpuz80.RegCondM:

		return true
	default:
		return false
	}
}

func isIdentifierOperand(operand rawOperand, value string) bool {
	if operand.parenthesized || operand.token.Type != token.Identifier {
		return false
	}

	return strings.EqualFold(operand.token.Value, value)
}

func hasParenthesizedValueOperand(operand rawOperand) bool {
	if !operand.parenthesized || operand.displacement != nil {
		return false
	}

	_, ok, err := operandValue(operand)
	return err == nil && ok
}

func hasPlainValueOperand(operand rawOperand) bool {
	if operand.parenthesized || operand.displacement != nil {
		return false
	}

	_, ok, err := operandValue(operand)
	return err == nil && ok
}
