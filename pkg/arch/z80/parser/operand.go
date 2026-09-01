package parser

import (
	"errors"
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
)

var (
	errInvalidOperandKind     = errors.New("invalid Z80 operand kind")
	errInvalidOperandRegister = errors.New("invalid Z80 operand register")
	errInvalidOperandValue    = errors.New("invalid Z80 operand value")
)

// OperandKind identifies the syntax and semantics of one Z80 operand.
type OperandKind uint8

const (
	OperandInvalid OperandKind = iota
	OperandRegister
	OperandValue
	OperandIndirectRegister
	OperandIndirectValue
	OperandIndexed
)

// Operand retains a resolved instruction's source-level operand shape. Value
// contains the immediate, address, label, expression, or indexed displacement.
type Operand struct {
	Kind     OperandKind
	Register cpuz80.RegisterParam
	Value    ast.Node
}

// Operands is the typed operand list accepted by the Z80 codec builder.
type Operands []Operand

// RegisterOperand constructs a direct register or condition operand.
func RegisterOperand(register cpuz80.RegisterParam) Operand {
	return Operand{Kind: OperandRegister, Register: register}
}

// ValueOperand constructs an immediate, address, label, or expression operand.
func ValueOperand(value ast.Node) Operand {
	return Operand{Kind: OperandValue, Value: value}
}

// IndirectRegisterOperand constructs a parenthesized register operand.
func IndirectRegisterOperand(register cpuz80.RegisterParam) Operand {
	return Operand{Kind: OperandIndirectRegister, Register: register}
}

// IndirectValueOperand constructs a parenthesized address or expression operand.
func IndirectValueOperand(value ast.Node) Operand {
	return Operand{Kind: OperandIndirectValue, Value: value}
}

// IndexedOperand constructs an IX/IY indirect operand with a displacement.
func IndexedOperand(register cpuz80.RegisterParam, displacement ast.Node) Operand {
	return Operand{Kind: OperandIndexed, Register: register, Value: displacement}
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

func operandsFromRaw(operands []rawOperand, resolvedValues []ast.Node) ([]Operand, error) {
	if operands == nil {
		return nil, nil
	}
	result := make([]Operand, len(operands))
	for index, operand := range operands {
		converted, err := operandFromRaw(operand, resolvedValues)
		if err != nil {
			return nil, fmt.Errorf("operand %d: %w", index, err)
		}
		result[index] = converted
	}
	return result, nil
}

func operandFromRaw(operand rawOperand, resolvedValues []ast.Node) (Operand, error) {
	if operand.displacement != nil {
		register, ok := operandIndexedRegister(operand)
		if !ok {
			return Operand{}, errInvalidOperandRegister
		}
		return IndexedOperand(register, operand.displacement.Copy()), nil
	}

	if operand.parenthesized {
		if value, ok, err := operandValue(operand); err != nil {
			return Operand{}, err
		} else if ok {
			return IndirectValueOperand(value.Copy()), nil
		}

		register, ok := sourceRegister(operand)
		if !ok {
			return Operand{}, errInvalidOperandRegister
		}
		return IndirectRegisterOperand(register), nil
	}

	if resolvedIdentifierValue, ok := resolvedIdentifierOperandValue(operand, resolvedValues); ok {
		return ValueOperand(resolvedIdentifierValue.Copy()), nil
	}
	if register, ok := sourceRegister(operand); ok {
		return RegisterOperand(register), nil
	}
	value, ok, err := operandValue(operand)
	if err != nil {
		return Operand{}, err
	}
	if !ok {
		return Operand{}, errInvalidOperandValue
	}
	return ValueOperand(value.Copy()), nil
}

func resolvedIdentifierOperandValue(operand rawOperand, resolvedValues []ast.Node) (ast.Node, bool) {
	if operand.token.Type != token.Identifier || operand.value != nil || operand.parenthesized {
		return nil, false
	}
	for _, value := range resolvedValues {
		if strings.EqualFold(ast.SymbolName(value), operand.token.Value) {
			return value, true
		}
	}
	return nil, false
}

func sourceRegister(operand rawOperand) (cpuz80.RegisterParam, bool) {
	if operand.token.Type == token.Identifier {
		if register, ok := registerOnlyCandidate(operand.token.Value); ok {
			return register, true
		}
		if condition, ok := conditionParamByName[strings.ToLower(operand.token.Value)]; ok {
			return condition, true
		}
	}

	if len(operand.registerParams) == 0 {
		return cpuz80.RegNone, false
	}
	return sourceRegisterParam(operand.registerParams[0]), true
}

func sourceRegisterParam(register cpuz80.RegisterParam) cpuz80.RegisterParam {
	switch register {
	case cpuz80.RegBCIndirect:
		return cpuz80.RegBC
	case cpuz80.RegDEIndirect:
		return cpuz80.RegDE
	case cpuz80.RegHLIndirect:
		return cpuz80.RegHL
	case cpuz80.RegSPIndirect:
		return cpuz80.RegSP
	default:
		return register
	}
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

func (operand Operand) raw() (rawOperand, error) {
	switch operand.Kind {
	case OperandRegister:
		if !validRegisterParam(operand.Register) {
			return rawOperand{}, errInvalidOperandRegister
		}
		return rawOperand{token: identifierToken(operand.Register)}, nil

	case OperandValue:
		if operand.Value == nil {
			return rawOperand{}, errInvalidOperandValue
		}
		return rawOperand{value: operand.Value.Copy()}, nil

	case OperandIndirectRegister:
		register := sourceRegisterParam(operand.Register)
		if !validRegisterParam(register) {
			return rawOperand{}, errInvalidOperandRegister
		}
		candidates := registerCandidatesForIndirectIdentifier(register.String())
		if len(candidates) == 0 {
			return rawOperand{}, errInvalidOperandRegister
		}
		return rawOperand{parenthesized: true, registerParams: candidates}, nil

	case OperandIndirectValue:
		if operand.Value == nil {
			return rawOperand{}, errInvalidOperandValue
		}
		return rawOperand{parenthesized: true, value: operand.Value.Copy()}, nil

	case OperandIndexed:
		if operand.Value == nil {
			return rawOperand{}, errInvalidOperandValue
		}
		register := indexedRegisterParam(operand.Register)
		if register == cpuz80.RegNone {
			return rawOperand{}, errInvalidOperandRegister
		}
		return rawOperand{
			displacement:   operand.Value.Copy(),
			parenthesized:  true,
			registerParams: []cpuz80.RegisterParam{register},
		}, nil

	default:
		return rawOperand{}, errInvalidOperandKind
	}
}

func identifierToken(register cpuz80.RegisterParam) token.Token {
	return token.Token{Type: token.Identifier, Value: register.String()}
}

func validRegisterParam(register cpuz80.RegisterParam) bool {
	return register != cpuz80.RegNone && register.String() != "unknown"
}

func indexedRegisterParam(register cpuz80.RegisterParam) cpuz80.RegisterParam {
	switch register {
	case cpuz80.RegIX, cpuz80.RegIXIndirect:
		return cpuz80.RegIXIndirect
	case cpuz80.RegIY, cpuz80.RegIYIndirect:
		return cpuz80.RegIYIndirect
	default:
		return cpuz80.RegNone
	}
}
