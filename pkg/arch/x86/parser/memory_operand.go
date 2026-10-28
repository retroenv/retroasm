package parser

import (
	"fmt"
	"strconv"

	"github.com/retroenv/retroasm/pkg/parser/ast"
	isa "github.com/retroenv/retrogolib/arch/cpu/x86"
)

// MemoryOperand retains an 8086 effective address. Address is RegMem for a
// direct address, or one of the eight canonical Reg*Ref forms. Width is zero
// for inference, one for a byte, or two for a word. Segment zero uses the
// default segment; other values must be segment registers. Symbol and
// Displacement form the address displacement before register addition.
type MemoryOperand struct {
	Address      isa.RegisterParam
	Width        uint8
	Segment      isa.RegisterParam
	Symbol       string
	Displacement int64
}

// SignedImmediate retains a signed literal until the operand width is known.
type SignedImmediate int64

// RepeatedString identifies the string operation after a repeat prefix.
type RepeatedString string

// InstructionFormKey retains all fields that can change the encoded address.
func (operand MemoryOperand) InstructionFormKey() string {
	return fmt.Sprintf("x86-memory/%d/%d/%d/%s/%d", operand.Address, operand.Width,
		operand.Segment, operand.Symbol, operand.Displacement)
}

// InstructionReferences exposes the displacement symbol to shared AST tools.
func (operand MemoryOperand) InstructionReferences() []ast.InstructionReference {
	if operand.Symbol == "" {
		return nil
	}
	return []ast.InstructionReference{{Value: ast.NewLabel(operand.Symbol), ReferenceType: ast.FullAddress}}
}

// RewriteInstructionSymbols changes only the displacement symbol.
func (operand MemoryOperand) RewriteInstructionSymbols(rename func(string) string) (any, error) {
	if operand.Symbol != "" {
		operand.Symbol = rename(operand.Symbol)
	}

	return operand, nil
}

// InstructionFormKey identifies the complete signed literal.
func (value SignedImmediate) InstructionFormKey() string {
	return "x86-immediate/" + strconv.FormatInt(int64(value), 10)
}

// InstructionReferences reports that a literal has no symbols.
func (SignedImmediate) InstructionReferences() []ast.InstructionReference { return nil }

// RewriteInstructionSymbols retains the literal.
func (value SignedImmediate) RewriteInstructionSymbols(_ func(string) string) (any, error) {
	return value, nil
}

// InstructionFormKey identifies the repeated string operation.
func (value RepeatedString) InstructionFormKey() string { return "x86-repeat/" + string(value) }

// InstructionReferences reports that the mnemonic has no symbols.
func (RepeatedString) InstructionReferences() []ast.InstructionReference { return nil }

// RewriteInstructionSymbols retains the mnemonic.
func (value RepeatedString) RewriteInstructionSymbols(_ func(string) string) (any, error) {
	return value, nil
}

func formatMemory(operand MemoryOperand) (string, error) {
	text := ""
	switch operand.Width {
	case 0:
	case 1:
		text = "byte "
	case 2:
		text = "word "
	default:
		return "", fmt.Errorf("invalid 8086 memory width %d", operand.Width)
	}

	text += "["
	if operand.Segment != 0 {
		if !operand.Segment.IsSegment() {
			return "", fmt.Errorf("invalid 8086 segment %v", operand.Segment)
		}
		text += operand.Segment.String() + ":"
	}

	address, ok := memoryAddressText(operand.Address)
	if !ok {
		return "", fmt.Errorf("invalid 8086 effective address %v", operand.Address)
	}
	if operand.Symbol != "" {
		if address != "" {
			address += "+"
		}

		address += operand.Symbol
	}

	if operand.Displacement > 0 && address != "" {
		address += "+"
	}

	if operand.Displacement != 0 || address == "" {
		address += strconv.FormatInt(operand.Displacement, 10)
	}

	return text + address + "]", nil
}

func memoryAddressText(address isa.RegisterParam) (string, bool) {
	if address == isa.RegMem {
		return "", true
	}
	forms := [...]string{"bx+si", "bx+di", "bp+si", "bp+di", "si", "di", "bp", "bx"}
	if address < isa.RegBXSIRef || address > isa.RegBXRef {
		return "", false
	}
	return forms[address-isa.RegBXSIRef], true
}
