package ast

import (
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

func TestInstructionArgument_Copy(t *testing.T) {
	original := NewInstructionArgument(uint16(0x1234))

	copied, ok := original.Copy().(InstructionArgument)
	assert.True(t, ok)
	assert.Equal(t, uint16(0x1234), copied.Value)
}

func TestInstructionArguments_Copy(t *testing.T) {
	original := NewInstructionArguments(
		NewNumber(1),
		NewLabel("target"),
		NewInstructionArgument("register"),
	)

	copied, ok := original.Copy().(InstructionArguments)
	assert.True(t, ok)
	assert.Len(t, copied.Values, 3)

	_, ok = copied.Values[0].(Number)
	assert.True(t, ok)

	_, ok = copied.Values[1].(Label)
	assert.True(t, ok)

	typedArg, ok := copied.Values[2].(InstructionArgument)
	assert.True(t, ok)
	assert.Equal(t, "register", typedArg.Value)
}
