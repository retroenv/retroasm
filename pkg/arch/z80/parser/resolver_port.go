package parser

import (
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
)

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

func isPortCOperand(operand rawOperand) bool {
	if !operand.parenthesized || operand.value != nil || operand.displacement != nil {
		return false
	}

	return containsRegisterParam(operandRegisterCandidates(operand), cpuz80.RegC)
}
