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
	"github.com/retroenv/retrogolib/set"
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

func resolveTwoOperands(variants []*cpuz80.Instruction, operand1, operand2 rawOperand) (*ResolvedInstruction, error) {
	if result := resolveRegisterPairOperands(variants, operand1, operand2); result != nil {
		return result, nil
	}
	if result := resolveAluRegisterPairOperands(variants, operand1, operand2); result != nil {
		return result, nil
	}
	if result := resolveIndirectLoadStoreOperands(variants, operand1, operand2); result != nil {
		return result, nil
	}

	if result, matched, err := resolveIndirectImmediateOperands(variants, operand1, operand2); err != nil || matched {
		return result, err
	}
	if result := resolveSpecialRegisterPairOperands(variants, operand1, operand2); result != nil {
		return result, nil
	}

	return resolveTwoOperandsFallback(variants, operand1, operand2)
}

func resolveTwoOperandsFallback(variants []*cpuz80.Instruction, operand1, operand2 rawOperand) (*ResolvedInstruction, error) {
	if result, matched, err := resolveExtendedRegisterMemoryOperands(variants, operand1, operand2); err != nil || matched {
		return result, err
	}

	if result, matched := resolvePortRegisterOperands(variants, operand1, operand2); matched {
		return result, nil
	}
	if result, matched, err := resolvePortImmediateOperands(variants, operand1, operand2); err != nil || matched {
		return result, err
	}

	if result, matched := resolveRegisterIndexedOperands(variants, operand1, operand2); matched {
		return result, nil
	}
	if result, matched := resolveIndexedRegisterOperands(variants, operand1, operand2); matched {
		return result, nil
	}

	if result, matched, err := resolveRegisterValueOperands(variants, operand1, operand2); err != nil || matched {
		return result, err
	}
	if result, matched, err := resolveValueRegisterOperands(variants, operand1, operand2); err != nil || matched {
		return result, err
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

// resolveAluRegisterPairOperands handles two-register ALU operations where the
// second operand is the RegisterOpcodes key (e.g., ADD A,B; ADD HL,BC; ADD IX,BC).
func resolveAluRegisterPairOperands(variants []*cpuz80.Instruction, operand1, operand2 rawOperand) *ResolvedInstruction {
	if operand1.parenthesized || operand2.parenthesized || operand1.displacement != nil || operand2.displacement != nil {
		return nil
	}

	candidates1 := operandRegisterOnlyCandidates(operand1)
	candidates2 := operandRegisterOnlyCandidates(operand2)
	if len(candidates1) == 0 || len(candidates2) == 0 {
		return nil
	}

	for _, variant := range variants {
		if len(variant.RegisterOpcodes) == 0 || variant.Name == cpuz80.LdName {
			continue
		}

		for _, c2 := range candidates2 {
			opcodeInfo, ok := variant.RegisterOpcodes[c2]
			if !ok {
				continue
			}

			for _, c1 := range candidates1 {
				if !firstOperandMatchesPrefix(c1, opcodeInfo.Prefix) {
					continue
				}

				addressing := cpuz80.RegisterAddressing
				if variant.HasAddressing(cpuz80.RegisterAddressing) {
					addressing = cpuz80.RegisterAddressing
				}

				return &ResolvedInstruction{
					Addressing:     addressing,
					Instruction:    variant,
					RegisterParams: []cpuz80.RegisterParam{c2},
				}
			}
		}
	}

	return nil
}

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

// resolveSpecialRegisterPairOperands handles explicit register-pair patterns
// that cannot be resolved generically (e.g., LD SP,HL; LD I,A; LD A,I;
// LD R,A; LD A,R; EX DE,HL).
func resolveSpecialRegisterPairOperands(variants []*cpuz80.Instruction, operand1, operand2 rawOperand) *ResolvedInstruction {
	if operand1.parenthesized || operand2.parenthesized || operand1.displacement != nil || operand2.displacement != nil {
		return nil
	}

	candidates1 := operandRegisterOnlyCandidates(operand1)
	candidates2 := operandRegisterOnlyCandidates(operand2)
	if len(candidates1) == 0 || len(candidates2) == 0 {
		return nil
	}

	for _, c1 := range candidates1 {
		for _, c2 := range candidates2 {
			variant, ok := specialRegisterPairs[[2]cpuz80.RegisterParam{c1, c2}]
			if !ok {
				continue
			}
			if !containsVariant(variants, variant) {
				continue
			}

			addressing := cpuz80.RegisterAddressing
			if variant.HasAddressing(cpuz80.RegisterAddressing) {
				addressing = cpuz80.RegisterAddressing
			} else if variant.HasAddressing(cpuz80.ImpliedAddressing) {
				addressing = cpuz80.ImpliedAddressing
			}

			registerKey := c1
			if _, ok := variant.RegisterOpcodes[c1]; !ok {
				registerKey = c2
			}
			if _, ok := variant.RegisterOpcodes[registerKey]; !ok {
				return &ResolvedInstruction{
					Addressing:  addressing,
					Instruction: variant,
				}
			}

			return &ResolvedInstruction{
				Addressing:     addressing,
				Instruction:    variant,
				RegisterParams: []cpuz80.RegisterParam{registerKey},
			}
		}
	}

	return nil
}

var specialRegisterPairs = map[[2]cpuz80.RegisterParam]*cpuz80.Instruction{
	{cpuz80.RegI, cpuz80.RegA}:   cpuz80.EdLdIA,
	{cpuz80.RegR, cpuz80.RegA}:   cpuz80.EdLdRA,
	{cpuz80.RegA, cpuz80.RegI}:   cpuz80.EdLdAI,
	{cpuz80.RegA, cpuz80.RegR}:   cpuz80.EdLdAR,
	{cpuz80.RegSP, cpuz80.RegHL}: cpuz80.LdSp,
	{cpuz80.RegSP, cpuz80.RegIX}: cpuz80.DdLdSpIX,
	{cpuz80.RegSP, cpuz80.RegIY}: cpuz80.FdLdSpIY,
	{cpuz80.RegDE, cpuz80.RegHL}: cpuz80.ExDeHl,
	{cpuz80.RegAF, cpuz80.RegAF}: cpuz80.ExAf,
}

// firstOperandMatchesPrefix checks if a register used as the first operand
// in a two-register instruction matches the expected prefix for the variant.
func firstOperandMatchesPrefix(register cpuz80.RegisterParam, prefix byte) bool {
	switch register {
	case cpuz80.RegIX, cpuz80.RegIXIndirect:
		return prefix == cpuz80.PrefixDD
	case cpuz80.RegIY, cpuz80.RegIYIndirect:
		return prefix == cpuz80.PrefixFD
	default:
		return prefix == 0x00 || prefix == cpuz80.PrefixED
	}
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

func containsVariant(variants []*cpuz80.Instruction, target *cpuz80.Instruction) bool {
	return slices.Contains(variants, target)
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

			// For indexed+immediate (e.g., LD (IX+d),n), include displacement
			// before the immediate value in OperandValues.
			operandValues := make([]ast.Node, 0, 2)
			if operand1.displacement != nil {
				operandValues = append(operandValues, operand1.displacement)
			}
			operandValues = append(operandValues, value)

			// For ALU instructions (non-LD) where the first operand is the implicit
			// accumulator (RegA), omit RegisterParams so the opcode generator uses
			// the Addressing map (e.g., ADD A,n → 0xC6 not 0x87).
			var regParams []cpuz80.RegisterParam
			if candidate != cpuz80.RegA || addressing != cpuz80.ImmediateAddressing || variant.Name == cpuz80.LdName {
				regParams = []cpuz80.RegisterParam{candidate}
			}

			return &ResolvedInstruction{
				Addressing:     addressing,
				Instruction:    variant,
				RegisterParams: regParams,
				OperandValues:  operandValues,
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

func addressingFamilies(variants []*cpuz80.Instruction) set.Set[string] {
	families := set.New[string]()

	for _, variant := range variants {
		for addressing := range variant.Addressing {
			families.Add(addressingFamilyName(addressing))
		}
		if len(variant.RegisterOpcodes) > 0 || len(variant.RegisterPairOpcodes) > 0 {
			families.Add("register")
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
