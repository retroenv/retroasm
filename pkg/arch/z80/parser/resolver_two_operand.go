package parser

import (
	"slices"

	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
)

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

func containsVariant(variants []*cpuz80.Instruction, target *cpuz80.Instruction) bool {
	return slices.Contains(variants, target)
}
