// Package scope implements symbol scope handling.
package scope

import (
	"errors"
	"fmt"

	"github.com/retroenv/retroasm/pkg/lexer/token"
)

// ErrForwardReference is returned by Symbol.Value when a label's address has not
// been assigned yet — i.e. when code references a label that is defined later in
// the source file.  Callers that encounter this during an address-assignment pass
// should treat the reference as requiring absolute (widest) addressing and retry
// once all addresses have been assigned.
var ErrForwardReference = errors.New("label address not yet assigned")

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
	name        string
	address     uint64
	addressSet  bool // true once SetAddress has been called (distinguishes address 0 from unset)
	typ         SymbolType
	expression  Expression
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
		name:        sym.name,
		address:     sym.address,
		addressSet:  sym.addressSet,
		typ:         sym.typ,
		expression:  sym.expression.CopyExpression().(Expression),
	}
}

// SetAddress sets the address of the symbol. This is only useful for symbols of type label that
// gets referenced in code.
func (sym *Symbol) SetAddress(address uint64) {
	sym.address = address
	sym.addressSet = true
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
// or the value of the expression. The returned value can be of type int64, uint64 or []byte.
func (sym *Symbol) Value(scope *Scope) (any, error) {
	switch sym.typ {
	case AliasType, EquType:
		value, err := sym.expression.Evaluate(scope, 1)
		if err != nil {
			return 0, fmt.Errorf("getting symbol value: %w", err)
		}
		return value, nil

	case LabelType, FunctionType:
		if !sym.addressSet {
			return 0, ErrForwardReference
		}
		return sym.address, nil

	default:
		return 0, fmt.Errorf("unsupported symbol type %v", sym.typ)
	}
}
