package parser

import (
	"errors"
	"fmt"
	"slices"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/sm83"
)

var (
	errInvalidOperandKind     = errors.New("invalid SM83 operand kind")
	errInvalidOperandRegister = errors.New("invalid SM83 operand register")
	errInvalidOperandValue    = errors.New("invalid SM83 operand value")
)

// OperandKind identifies the syntax and semantics of one SM83 operand.
type OperandKind uint8

const (
	OperandInvalid OperandKind = iota
	OperandRegister
	OperandValue
	OperandIndirectRegister
	OperandIndirectValue
	OperandHLIncrement
	OperandHLDecrement
	OperandSPOffset
)

// Operand retains a resolved instruction's source-level operand shape.
type Operand struct {
	Kind     OperandKind
	Register sm83.RegisterParam
	Value    ast.Node
}

// Operands is the typed operand list accepted by the SM83 codec builder.
type Operands []Operand

func (operand Operand) raw() (rawOperand, error) {
	switch operand.Kind {
	case OperandRegister:
		return rawRegisterOperand(operand.Register)

	case OperandValue:
		if operand.Value == nil {
			return rawOperand{}, errInvalidOperandValue
		}
		return rawOperand{value: operand.Value.Copy()}, nil

	case OperandIndirectRegister:
		if !validIndirectRegister(operand.Register) {
			return rawOperand{}, errInvalidOperandRegister
		}
		if indirect := indirectRegister(operand.Register); indirect != sm83.RegNone {
			return rawOperand{indirect: true, indirectReg: indirect}, nil
		}
		return rawOperand{indirect: true, register: operand.Register}, nil

	case OperandIndirectValue:
		if operand.Value == nil {
			return rawOperand{}, errInvalidOperandValue
		}
		return rawOperand{indirect: true, value: operand.Value.Copy()}, nil

	case OperandHLIncrement:
		return rawOperand{indirect: true, isHLPlus: true}, nil

	case OperandHLDecrement:
		return rawOperand{indirect: true, isHLMinus: true}, nil

	case OperandSPOffset:
		if operand.Value == nil {
			return rawOperand{}, errInvalidOperandValue
		}
		return rawOperand{register: sm83.RegSP, value: operand.Value.Copy()}, nil

	default:
		return rawOperand{}, errInvalidOperandKind
	}
}

// RegisterOperand constructs a direct register or condition operand.
func RegisterOperand(register sm83.RegisterParam) Operand {
	return Operand{Kind: OperandRegister, Register: register}
}

// ValueOperand constructs an immediate, address, label, or expression operand.
func ValueOperand(value ast.Node) Operand {
	return Operand{Kind: OperandValue, Value: value}
}

// IndirectRegisterOperand constructs a parenthesized register operand.
func IndirectRegisterOperand(register sm83.RegisterParam) Operand {
	return Operand{Kind: OperandIndirectRegister, Register: register}
}

// IndirectValueOperand constructs a parenthesized address or expression operand.
func IndirectValueOperand(value ast.Node) Operand {
	return Operand{Kind: OperandIndirectValue, Value: value}
}

// HLIncrementOperand constructs the SM83 post-increment memory operand (HL+).
func HLIncrementOperand() Operand {
	return Operand{Kind: OperandHLIncrement}
}

// HLDecrementOperand constructs the SM83 post-decrement memory operand (HL-).
func HLDecrementOperand() Operand {
	return Operand{Kind: OperandHLDecrement}
}

// SPOffsetOperand constructs the signed-byte SP+e operand.
func SPOffsetOperand(offset ast.Node) Operand {
	return Operand{Kind: OperandSPOffset, Register: sm83.RegSP, Value: offset}
}

func copyOperands(operands []Operand) []Operand {
	if operands == nil {
		return nil
	}
	copied := make([]Operand, len(operands))
	for index, operand := range operands {
		copied[index] = operand
		if operand.Value != nil {
			copied[index].Value = operand.Value.Copy()
		}
	}
	return copied
}

func operandsFromRaw(operands []rawOperand, resolved *ResolvedInstruction) ([]Operand, error) {
	if operands == nil {
		return nil, nil
	}
	result := make([]Operand, len(operands))
	for index, operand := range operands {
		converted, err := operandFromRaw(operand, resolved)
		if err != nil {
			return nil, fmt.Errorf("operand %d: %w", index, err)
		}
		result[index] = converted
	}
	return result, nil
}

func operandFromRaw(operand rawOperand, resolved *ResolvedInstruction) (Operand, error) {
	switch {
	case operand.isHLPlus:
		return HLIncrementOperand(), nil
	case operand.isHLMinus:
		return HLDecrementOperand(), nil
	case operand.register == sm83.RegSP && operand.value != nil && !operand.indirect:
		return SPOffsetOperand(operand.value.Copy()), nil
	case operand.indirect:
		return indirectOperandFromRaw(operand)
	case operand.isCondition:
		return conditionOperandFromRaw(operand, resolved), nil
	case operand.register != sm83.RegNone:
		return RegisterOperand(operand.register), nil
	default:
		value, ok, err := operandValue(operand)
		if err != nil {
			return Operand{}, err
		}
		if !ok {
			return Operand{}, errInvalidOperandValue
		}
		return ValueOperand(value.Copy()), nil
	}
}

func conditionOperandFromRaw(operand rawOperand, resolved *ResolvedInstruction) Operand {
	condition := operand.register
	if isCondition(operand.indirectReg) &&
		(operand.register != sm83.RegC || slices.Contains(resolved.RegisterParams, operand.indirectReg)) {

		condition = operand.indirectReg
	}
	return RegisterOperand(condition)
}

func indirectOperandFromRaw(operand rawOperand) (Operand, error) {
	if operand.value != nil {
		return IndirectValueOperand(operand.value.Copy()), nil
	}
	register := operand.register
	if register == sm83.RegNone {
		register = sourceIndirectRegister(operand.indirectReg)
	}
	if !validIndirectRegister(register) {
		return Operand{}, errInvalidOperandRegister
	}
	return IndirectRegisterOperand(register), nil
}

func rawOperands(operands []Operand) ([]rawOperand, error) {
	result := make([]rawOperand, len(operands))
	for index, operand := range operands {
		converted, err := operand.raw()
		if err != nil {
			return nil, fmt.Errorf("operand %d: %w", index, err)
		}
		result[index] = converted
	}
	return result, nil
}

func rawRegisterOperand(register sm83.RegisterParam) (rawOperand, error) {
	if !validDirectRegister(register) {
		return rawOperand{}, errInvalidOperandRegister
	}
	if register == sm83.RegCondC {
		return rawOperand{
			token: identifierToken(register), register: sm83.RegC,
			indirectReg: register, isCondition: true,
		}, nil
	}
	return rawOperand{
		token: identifierToken(register), register: register,
		isCondition: isCondition(register),
	}, nil
}

func identifierToken(register sm83.RegisterParam) token.Token {
	return token.Token{Type: token.Identifier, Value: register.String()}
}

func validDirectRegister(register sm83.RegisterParam) bool {
	switch register {
	case sm83.RegA, sm83.RegB, sm83.RegC, sm83.RegD, sm83.RegE, sm83.RegH, sm83.RegL,
		sm83.RegAF, sm83.RegBC, sm83.RegDE, sm83.RegHL, sm83.RegSP,
		sm83.RegCondNZ, sm83.RegCondZ, sm83.RegCondNC, sm83.RegCondC:
		return true
	default:
		return false
	}
}

func validIndirectRegister(register sm83.RegisterParam) bool {
	return register == sm83.RegBC || register == sm83.RegDE ||
		register == sm83.RegHL || register == sm83.RegC
}

func indirectRegister(register sm83.RegisterParam) sm83.RegisterParam {
	switch register {
	case sm83.RegBC:
		return sm83.RegBCIndirect
	case sm83.RegDE:
		return sm83.RegDEIndirect
	case sm83.RegHL:
		return sm83.RegHLIndirect
	default:
		return sm83.RegNone
	}
}

func sourceIndirectRegister(register sm83.RegisterParam) sm83.RegisterParam {
	switch register {
	case sm83.RegBCIndirect:
		return sm83.RegBC
	case sm83.RegDEIndirect:
		return sm83.RegDE
	case sm83.RegHLIndirect:
		return sm83.RegHL
	default:
		return register
	}
}
