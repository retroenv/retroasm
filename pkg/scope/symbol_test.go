package scope

import (
	"errors"
	"testing"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retrogolib/assert"
)

type mockExpr struct {
	value any
	err   error
}

func (m *mockExpr) CopyExpression() any                   { return m }
func (m *mockExpr) Evaluate(_ *Scope, _ int) (any, error) { return m.value, m.err }
func (m *mockExpr) EvaluateAtProgramCounter(_ *Scope, _ int, _ uint64) (any, error) {
	return m.value, m.err
}
func (m *mockExpr) IsEvaluatedAtAddressAssign() bool { return false }
func (m *mockExpr) IsEvaluatedOnce() bool            { return false }
func (m *mockExpr) Tokens() []token.Token            { return nil }

func TestScopeParent(t *testing.T) {
	parent := New(nil)
	assert.Nil(t, parent.Parent())
	child := New(parent)
	assert.Equal(t, parent, child.Parent())
}

func TestScopeAllLabels(t *testing.T) {
	sc := New(nil)
	for _, sym := range []*Symbol{
		{name: "lbl", typ: LabelType, address: 0x100},
		{name: "fn", typ: FunctionType, address: 0x200},
		{name: "equ", typ: EquType},
	} {
		assert.NoError(t, sc.AddSymbol(sym))
	}
	labels := sc.AllLabels()
	assert.Len(t, labels, 2)
	assert.Equal(t, uint64(0x100), labels["lbl"])
	assert.Equal(t, uint64(0x200), labels["fn"])
}

func TestScopeAddSymbolAliasOverwrite(t *testing.T) {
	sc := New(nil)
	assert.NoError(t, sc.AddSymbol(&Symbol{name: "x", typ: AliasType}))
	assert.NoError(t, sc.AddSymbol(&Symbol{name: "x", typ: AliasType}))
	assert.Error(t, sc.AddSymbol(&Symbol{name: "x", typ: LabelType}))
}

func TestNewSymbol(t *testing.T) {
	sc := New(nil)
	sym, err := NewSymbol(sc, "foo", LabelType)
	assert.NoError(t, err)
	assert.Equal(t, LabelType, sym.Type())
	_, err = NewSymbol(sc, "foo", LabelType)
	assert.Error(t, err)
}

func TestSymbolValueLabel(t *testing.T) {
	sc := New(nil)
	for _, typ := range []SymbolType{LabelType, FunctionType} {
		sym := &Symbol{typ: typ}
		_, err := sym.Value(sc)
		assert.ErrorIs(t, err, ErrForwardReference)
		sym.SetAddress(0x300)
		val, err := sym.Value(sc)
		assert.NoError(t, err)
		assert.Equal(t, uint64(0x300), val)
	}
}

func TestSymbolValueExpression(t *testing.T) {
	sc := New(nil)
	for _, typ := range []SymbolType{AliasType, EquType} {
		sym := &Symbol{typ: typ, expression: &mockExpr{value: int64(42)}}
		val, err := sym.Value(sc)
		assert.NoError(t, err)
		assert.Equal(t, int64(42), val)

		sym2 := &Symbol{typ: typ, expression: &mockExpr{err: errors.New("bad")}}
		_, err = sym2.Value(sc)
		assert.Error(t, err)
	}
}

func TestSymbolValueInvalidType(t *testing.T) {
	sym := &Symbol{typ: invalidSymbolType}
	_, err := sym.Value(nil)
	assert.Error(t, err)
}

func TestSymbolExpressionAccessors(t *testing.T) {
	sym := &Symbol{}
	assert.Nil(t, sym.Expression())
	expr := &mockExpr{}
	sym.SetExpression(expr)
	assert.Equal(t, expr, sym.Expression())
}

func TestSymbolCopy(t *testing.T) {
	orig := &Symbol{
		name:       "orig",
		address:    0x500,
		addressSet: true,
		typ:        LabelType,
		expression: &mockExpr{value: int64(1)},
	}
	cp := orig.Copy()
	assert.Equal(t, orig.name, cp.name)
	assert.Equal(t, orig.address, cp.address)
	assert.Equal(t, orig.addressSet, cp.addressSet)
	assert.Equal(t, orig.typ, cp.typ)
}
