package scope

import (
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

func TestScope(t *testing.T) {
	parent := New(nil)
	sym := &Symbol{name: "test"}
	assert.NoError(t, parent.AddSymbol(sym))

	// add child with parent and get symbol that is defined in parent
	child := New(parent)
	found, err := child.GetSymbol(sym.name)
	assert.NoError(t, err)
	assert.Equal(t, sym, found)

	// add child for parent with same symbol
	symOverwrite := &Symbol{name: "test"}
	assert.NoError(t, child.AddSymbol(symOverwrite))
	found, err = child.GetSymbol(sym.name)
	assert.NoError(t, err)
	assert.Equal(t, symOverwrite, found)
	found, err = parent.GetSymbol(sym.name)
	assert.NoError(t, err)
	assert.Equal(t, sym, found)

	// adding symbol again fails
	err = parent.AddSymbol(sym)
	assert.Error(t, err)

	// getting an undefined symbol fails
	_, err = parent.GetSymbol("nonexisting")
	assert.Error(t, err)
}
