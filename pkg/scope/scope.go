package scope

import (
	"fmt"
)

// Scope defines a scope that contains symbols, on a global, file or function level.
// It supports embedding child scopes by a parent relationship.
type Scope struct {
	parent *Scope

	symbols map[string]*Symbol
}

// New creates a new scope with given parent that can be nil.
func New(parent *Scope) *Scope {
	return &Scope{
		parent:  parent,
		symbols: map[string]*Symbol{},
	}
}

// AddSymbol adds a symbol to the current scope.
func (sc *Scope) AddSymbol(sym *Symbol) error {
	existing, exists := sc.symbols[sym.name]
	if exists && (existing.typ != AliasType || sym.typ != AliasType) {
		return fmt.Errorf("symbol '%s' already exists and can not be overwritten", sym.name)
	}

	sc.symbols[sym.name] = sym
	return nil
}

// GetSymbol gets a symbol of the current scope, if it is not found in the current scope
// it traverses all parents to receive it.
func (sc *Scope) GetSymbol(name string) (*Symbol, error) {
	for lookup := sc; lookup != nil; lookup = lookup.parent {
		sym, ok := lookup.symbols[name]
		if ok {
			return sym, nil
		}
	}
	return nil, fmt.Errorf("symbol '%s' not found in scope", name)
}

// Parent returns the parent scope of the scope.
func (sc *Scope) Parent() *Scope {
	return sc.parent
}
