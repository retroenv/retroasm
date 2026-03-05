package parser

import (
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
)

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

	// For HL stores, prefer shorter Addressing-only encoding (e.g., LdExtended
	// opcode 0x22, 3 bytes) over ED-prefixed RegisterOpcodes (e.g., EdLdNnHl, 4 bytes).
	if containsRegisterParam(candidates, cpuz80.RegHL) {
		if result := resolveExtendedHLStore(variants, value); result != nil {
			return result, true, nil
		}
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

func resolveExtendedHLStore(variants []*cpuz80.Instruction, value ast.Node) *ResolvedInstruction {
	for _, variant := range variants {
		if !variant.HasAddressing(cpuz80.ExtendedAddressing) {
			continue
		}
		if _, hasRegHL := variant.RegisterOpcodes[cpuz80.RegHL]; hasRegHL {
			continue
		}

		addrInfo := variant.Addressing[cpuz80.ExtendedAddressing]
		if !matchesExtendedLoadDirection(variant, addrInfo.Opcode, false) {
			continue
		}

		return &ResolvedInstruction{
			Addressing:    cpuz80.ExtendedAddressing,
			Instruction:   variant,
			OperandValues: []ast.Node{value},
		}
	}
	return nil
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
