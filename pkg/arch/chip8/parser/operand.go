package parser

import "github.com/retroenv/retroasm/pkg/parser/ast"

// OperandKind identifies one CHIP-8 source operand shape.
type OperandKind uint8

const (
	OperandInvalid OperandKind = iota
	OperandRegister
	OperandByte
	OperandAddress
	OperandNibble
	OperandI
	OperandDT
	OperandST
	OperandF
	OperandB
	OperandK
	OperandIndirectI
)

// Operand retains one typed CHIP-8 source operand.
type Operand struct {
	Kind     OperandKind
	Register byte
	Value    ast.Node
}

// Operands is the CHIP-8 operand list accepted by the typed codec.
type Operands []Operand

// RegisterOperand constructs a V0..VF operand.
func RegisterOperand(register byte) Operand {
	return Operand{Kind: OperandRegister, Register: register}
}

// ByteOperand constructs an 8-bit value operand.
func ByteOperand(value ast.Node) Operand {
	return Operand{Kind: OperandByte, Value: value}
}

// AddressOperand constructs a 12-bit address operand.
func AddressOperand(value ast.Node) Operand {
	return Operand{Kind: OperandAddress, Value: value}
}

// NibbleOperand constructs a 4-bit value operand.
func NibbleOperand(value ast.Node) Operand {
	return Operand{Kind: OperandNibble, Value: value}
}

// SpecialOperand constructs an I, timer, key, font, BCD, or [I] operand.
func SpecialOperand(kind OperandKind) Operand {
	return Operand{Kind: kind}
}

func copyOperands(operands Operands) Operands {
	if operands == nil {
		return nil
	}
	copied := make(Operands, len(operands))
	for index, operand := range operands {
		copied[index] = operand
		if operand.Value != nil {
			copied[index].Value = operand.Value.Copy()
		}
	}
	return copied
}
