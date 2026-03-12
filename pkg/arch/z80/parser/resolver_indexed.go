package parser

import (
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
)

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

func matchesIndexedLoadDirection(variant *cpuz80.Instruction, opcode byte, registerFirst bool) bool {
	if variant.Name != cpuz80.LdName {
		return true
	}

	if registerFirst {
		return opcode&0x07 == 0x06
	}

	return opcode&0xF8 == 0x70
}
