package parser

import (
	"slices"

	"github.com/retroenv/retroasm/pkg/parser/ast"
)

// OperandKind identifies one CPU65816 source operand shape.
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
	OperandIndirectLong
	OperandIndirectLongIndexedY
	OperandStackRelative
	OperandStackRelativeIndirectIndexedY
	OperandBlockMoveBank
)

// AddressSize identifies an explicit direct-page, absolute, or long source qualifier.
type AddressSize uint8

const (
	AddressDefault AddressSize = iota
	AddressDirectPage
	AddressAbsolute
	AddressLong
)

// Operand retains one typed CPU65816 source operand.
type Operand struct {
	Kind      OperandKind
	Size      AddressSize
	Value     ast.Node
	Modifiers []ast.Modifier
}

// Operands is the CPU65816 operand list accepted by the typed codec.
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

// BlockMoveOperands constructs source and destination bank operands.
func BlockMoveOperands(source, destination ast.Node) Operands {
	return Operands{
		{Kind: OperandBlockMoveBank, Value: source},
		{Kind: OperandBlockMoveBank, Value: destination},
	}
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
