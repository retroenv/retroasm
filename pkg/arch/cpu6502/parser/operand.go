package parser

import (
	"slices"

	"github.com/retroenv/retroasm/pkg/parser/ast"
)

// OperandKind identifies one CPU6502 source operand shape.
type OperandKind uint8

const (
	OperandInvalid OperandKind = iota
	OperandAccumulator
	OperandImmediate
	OperandAddress
	OperandIndexedX
	OperandIndexedY
	OperandIndirect
	OperandIndexedXIndirect
	OperandIndirectIndexedY
	OperandRelativeTarget
)

// AddressSize identifies explicit zero-page or absolute source intent.
type AddressSize uint8

const (
	AddressDefault AddressSize = iota
	AddressZeroPage
	AddressAbsolute
)

// Operand retains one typed CPU6502 source operand.
type Operand struct {
	Kind      OperandKind
	Size      AddressSize
	Value     ast.Node
	Modifiers []ast.Modifier
}

// Operands is the CPU6502 operand list accepted by the typed codec.
type Operands []Operand

// AccumulatorOperand constructs the explicit A operand.
func AccumulatorOperand() Operand {
	return Operand{Kind: OperandAccumulator}
}

// ImmediateOperand constructs a #value operand.
func ImmediateOperand(value ast.Node) Operand {
	return Operand{Kind: OperandImmediate, Value: value}
}

// MemoryOperand constructs a value-bearing addressing operand.
func MemoryOperand(kind OperandKind, size AddressSize, value ast.Node) Operand {
	return Operand{Kind: kind, Size: size, Value: value}
}

// ZeroPageRelativeOperands constructs the two operands used by Rockwell-style
// BBR/BBS instructions.
func ZeroPageRelativeOperands(zeroPage, target ast.Node) Operands {
	return Operands{
		MemoryOperand(OperandAddress, AddressZeroPage, zeroPage),
		MemoryOperand(OperandRelativeTarget, AddressDefault, target),
	}
}

// WithModifiers returns an operand with a private copy of address modifiers.
func WithModifiers(operand Operand, modifiers ...ast.Modifier) Operand {
	operand.Modifiers = slices.Clone(modifiers)
	return operand
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
		copied[index].Modifiers = slices.Clone(operand.Modifiers)
	}
	return copied
}
