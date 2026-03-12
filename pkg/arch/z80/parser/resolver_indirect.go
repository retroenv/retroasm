package parser

import (
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
)

// resolveIndirectLoadStoreOperands handles patterns where one operand is a
// parenthesized indirect register and the other is a direct register
// (e.g., LD A,(HL); LD (HL),A; LD A,(BC); LD (BC),A; EX (SP),IX).
func resolveIndirectLoadStoreOperands(variants []*cpuz80.Instruction, operand1, operand2 rawOperand) *ResolvedInstruction {
	if operand1.displacement != nil || operand2.displacement != nil {
		return nil
	}

	var regOp, indOp rawOperand
	var isLoad bool
	switch {
	case operand2.parenthesized && !operand1.parenthesized:
		regOp, indOp = operand1, operand2
		isLoad = true
	case operand1.parenthesized && !operand2.parenthesized:
		regOp, indOp = operand2, operand1
		isLoad = false
	default:
		return nil
	}

	regCandidates := operandRegisterOnlyCandidates(regOp)
	indCandidates := operandRegisterCandidates(indOp)
	if len(regCandidates) == 0 || len(indCandidates) == 0 {
		return nil
	}

	// Skip if the indirect operand is (c) — handled by port register resolver.
	if !hasIndirectRegisterParam(indCandidates) {
		return nil
	}

	keys := indirectRegisterKeys(regCandidates, indCandidates, isLoad)
	if result := matchIndirectLoadStoreKeys(variants, keys); result != nil {
		return result
	}

	// Fallback: try RegisterPairOpcodes for indirect store operations
	// (e.g., LD (HL),A uses RegisterPairOpcodes[{RegHLIndirect, RegA}] in LdReg8).
	pairKeys := indirectRegisterPairKeys(regCandidates, indCandidates, isLoad)
	if result := matchIndirectLoadStorePairKeys(variants, pairKeys); result != nil {
		return result
	}

	// Fallback: Addressing-only variants for EX (SP),HL/IX/IY.
	// Only match when the indirect operand is (sp).
	if containsRegisterParam(indCandidates, cpuz80.RegSPIndirect) {
		return resolveStackPointerIndirectVariant(variants)
	}

	return nil
}

func matchIndirectLoadStoreKeys(variants []*cpuz80.Instruction, keys []cpuz80.RegisterParam) *ResolvedInstruction {
	for _, variant := range variants {
		if len(variant.RegisterOpcodes) == 0 {
			continue
		}

		for _, key := range keys {
			if _, ok := variant.RegisterOpcodes[key]; !ok {
				continue
			}

			return &ResolvedInstruction{
				Addressing:     indirectLoadStoreAddressing(variant),
				Instruction:    variant,
				RegisterParams: []cpuz80.RegisterParam{key},
			}
		}
	}
	return nil
}

func indirectRegisterPairKeys(regCandidates, indCandidates []cpuz80.RegisterParam, isLoad bool) [][2]cpuz80.RegisterParam {
	var keys [][2]cpuz80.RegisterParam
	for _, ind := range indCandidates {
		for _, reg := range regCandidates {
			if isLoad {
				keys = append(keys, [2]cpuz80.RegisterParam{reg, ind})
			} else {
				keys = append(keys, [2]cpuz80.RegisterParam{ind, reg})
			}
		}
	}
	return keys
}

func matchIndirectLoadStorePairKeys(variants []*cpuz80.Instruction, keys [][2]cpuz80.RegisterParam) *ResolvedInstruction {
	for _, variant := range variants {
		if len(variant.RegisterPairOpcodes) == 0 {
			continue
		}

		for _, key := range keys {
			if _, ok := variant.RegisterPairOpcodes[key]; !ok {
				continue
			}

			return &ResolvedInstruction{
				Addressing:     indirectLoadStoreAddressing(variant),
				Instruction:    variant,
				RegisterParams: []cpuz80.RegisterParam{key[0], key[1]},
			}
		}
	}
	return nil
}

func resolveStackPointerIndirectVariant(variants []*cpuz80.Instruction) *ResolvedInstruction {
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

func indirectLoadStoreAddressing(variant *cpuz80.Instruction) cpuz80.AddressingMode {
	if variant.HasAddressing(cpuz80.RegisterIndirectAddressing) {
		return cpuz80.RegisterIndirectAddressing
	}
	if variant.HasAddressing(cpuz80.RegisterAddressing) {
		return cpuz80.RegisterAddressing
	}
	return cpuz80.RegisterIndirectAddressing
}

// resolveIndirectImmediateOperands handles instructions where operand1 is a
// parenthesized register (or indexed) and operand2 is an immediate value
// (e.g., LD (HL),n; LD (IX+d),n; LD (IY+d),n).
func resolveIndirectImmediateOperands(variants []*cpuz80.Instruction, operand1, operand2 rawOperand) (*ResolvedInstruction, bool, error) {
	if !operand1.parenthesized {
		return nil, false, nil
	}

	// Skip if operand2 is a register, not an immediate value.
	if len(operandRegisterCandidates(operand2)) > 0 {
		return nil, false, nil
	}

	value, ok, err := operandValue(operand2)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}

	// Case 1: LD (HL),n — uses RegisterIndirectAddressing without RegisterOpcodes.
	if operand1.displacement == nil {
		if result := resolveIndirectRegisterImmediate(variants, value); result != nil {
			return result, true, nil
		}
		return nil, false, nil
	}

	// Case 2: LD (IX+d),n / LD (IY+d),n — uses ImmediateAddressing with displacement.
	result := resolveIndexedImmediate(variants, operand1, value)
	return result, result != nil, nil
}

func resolveIndirectRegisterImmediate(variants []*cpuz80.Instruction, value ast.Node) *ResolvedInstruction {
	for _, variant := range variants {
		if len(variant.RegisterOpcodes) != 0 {
			continue
		}
		if !variant.HasAddressing(cpuz80.RegisterIndirectAddressing) {
			continue
		}

		return &ResolvedInstruction{
			Addressing:    cpuz80.RegisterIndirectAddressing,
			Instruction:   variant,
			OperandValues: []ast.Node{value},
		}
	}
	return nil
}

func resolveIndexedImmediate(variants []*cpuz80.Instruction, operand1 rawOperand, value ast.Node) *ResolvedInstruction {
	indexedRegister, ok := operandIndexedRegister(operand1)
	if !ok {
		return nil
	}

	for _, variant := range variants {
		if !variant.HasAddressing(cpuz80.ImmediateAddressing) {
			continue
		}

		for param, opcodeInfo := range variant.RegisterOpcodes {
			if !matchesIndexedRegisterPrefix(indexedRegister, opcodeInfo.Prefix) {
				continue
			}
			// Only match displacement+immediate variants (keyed by RegImm8
			// or RegIYIndirect), not 16-bit immediate loads (keyed by RegIX/RegIY).
			if param == cpuz80.RegIX || param == cpuz80.RegIY {
				continue
			}

			return &ResolvedInstruction{
				Addressing:     cpuz80.ImmediateAddressing,
				Instruction:    variant,
				RegisterParams: []cpuz80.RegisterParam{param},
				OperandValues:  []ast.Node{operand1.displacement, value},
			}
		}
	}

	return nil
}

// indirectRegisterKeys builds candidate RegisterOpcodes keys for an indirect
// register operation based on the register operands and direction.
func indirectRegisterKeys(regCandidates, indCandidates []cpuz80.RegisterParam, isLoad bool) []cpuz80.RegisterParam {
	var keys []cpuz80.RegisterParam

	for _, reg := range regCandidates {
		for _, ind := range indCandidates {
			if isLoad {
				if mapped, ok := hlLoadRegisterParam(reg, ind); ok {
					keys = append(keys, mapped)
				}
			} else {
				if ind == cpuz80.RegHLIndirect {
					keys = append(keys, reg)
				}
				keys = append(keys, ind)
				keys = append(keys, reg)
			}
		}
	}

	return keys
}

// hlLoadRegisterParam maps a destination register and indirect source to
// the special RegisterParam used in LdReg8/LdIndirect RegisterOpcodes.
func hlLoadRegisterParam(reg, ind cpuz80.RegisterParam) (cpuz80.RegisterParam, bool) {
	if ind == cpuz80.RegHLIndirect {
		switch reg {
		case cpuz80.RegB:
			return cpuz80.RegLoadHLB, true
		case cpuz80.RegC:
			return cpuz80.RegLoadHLC, true
		case cpuz80.RegD:
			return cpuz80.RegLoadHLD, true
		case cpuz80.RegE:
			return cpuz80.RegLoadHLE, true
		case cpuz80.RegH:
			return cpuz80.RegLoadHLH, true
		case cpuz80.RegL:
			return cpuz80.RegLoadHLL, true
		case cpuz80.RegA:
			return cpuz80.RegLoadHLA, true
		}
	}
	if ind == cpuz80.RegBCIndirect && reg == cpuz80.RegA {
		return cpuz80.RegLoadBC, true
	}
	if ind == cpuz80.RegDEIndirect && reg == cpuz80.RegA {
		return cpuz80.RegLoadDE, true
	}

	return cpuz80.RegNone, false
}

// hasIndirectRegisterParam returns true if any candidate is a known indirect
// register (HLIndirect, BCIndirect, DEIndirect, SPIndirect, IX, IY).
func hasIndirectRegisterParam(candidates []cpuz80.RegisterParam) bool {
	for _, c := range candidates {
		switch c {
		case cpuz80.RegHLIndirect, cpuz80.RegBCIndirect, cpuz80.RegDEIndirect,
			cpuz80.RegSPIndirect, cpuz80.RegIX, cpuz80.RegIY:
			return true
		}
	}
	return false
}
