// Package scope implements symbol scope handling.
package scope

import (
	"fmt"

	"github.com/retroenv/retroasm/lexer/token"
)

// SymbolType defines a type of symbol.
type SymbolType int

const (
	invalidSymbolType SymbolType = iota
	AliasType
	EquType
	LabelType
	FunctionType
)

// Expression defines the used expression functions.
type Expression interface {
	CopyExpression() any
	Evaluate(scope *Scope, dataWidth int) (any, error)
	EvaluateAtProgramCounter(scope *Scope, dataWidth int, programCounter uint64) (any, error)
	IsEvaluatedAtAddressAssign() bool
	IsEvaluatedOnce() bool
	Tokens() []token.Token
}

// Symbol defines a symbol that is part of a scope.
type Symbol struct {
	name       string
	address    uint64
	typ        SymbolType
	expression Expression
}

// NewSymbol creates a new symbol in the given scope.
func NewSymbol(scope *Scope, name string, typ SymbolType) (*Symbol, error) {
	sym := &Symbol{
		name: name,
		typ:  typ,
	}

	if err := scope.AddSymbol(sym); err != nil {
		return nil, fmt.Errorf("adding symbol: %w", err)
	}
	return sym, nil
}

// Copy returns a copy of the symbol.
func (sym *Symbol) Copy() *Symbol {
	return &Symbol{
		name:       sym.name,
		address:    sym.address,
		typ:        sym.typ,
		expression: sym.expression.CopyExpression().(Expression),
	}
}

// SetAddress sets the address of the symbol. This is only useful for symbols of type label that
// gets referenced in code.
func (sym *Symbol) SetAddress(address uint64) {
	sym.address = address
}

// SetExpression sets the expression of the symbol.
func (sym *Symbol) SetExpression(expression Expression) {
	sym.expression = expression
}

// Expression returns the expression of the symbol.
func (sym *Symbol) Expression() Expression {
	return sym.expression
}

// Type returns the type of the symbol.
func (sym *Symbol) Type() SymbolType {
	return sym.typ
}

// Value returns the value of the symbol, either an address for symbols of type label
// or the value of the expression. The returned value can be of can be of type int64,
// uint64 or []byte.
func (sym *Symbol) Value(scope *Scope) (any, error) {
	switch sym.typ {
	case AliasType, EquType:
		value, err := sym.expression.Evaluate(scope, 1)
		if err != nil {
			return 0, fmt.Errorf("getting symbol value: %w", err)
		}
		return value, nil

	case LabelType, FunctionType:
		return sym.address, nil

	default:
		return 0, fmt.Errorf("unsupported symbol type %v", sym.typ)
	}
}
