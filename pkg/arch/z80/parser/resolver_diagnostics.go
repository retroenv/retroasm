package parser

import (
	"fmt"
	"slices"
	"strings"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
	"github.com/retroenv/retrogolib/set"
)

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
