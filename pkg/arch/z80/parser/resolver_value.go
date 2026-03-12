package parser

import (
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
)

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
