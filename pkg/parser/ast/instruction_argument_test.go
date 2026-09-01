package ast

import (
	"reflect"
	"slices"
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

type mutableInstructionArgument struct {
	values []int
}

type immutableInstructionArgument struct {
	name   string
	values [2]uint16
}

type unsupportedMutableInstructionArgument struct {
	values []int
}

func (argument mutableInstructionArgument) CopyInstructionArgument() any {
	argument.values = slices.Clone(argument.values)
	return argument
}

func TestInstructionArgument_Copy(t *testing.T) {
	original := NewInstructionArgument(immutableInstructionArgument{
		name:   "register",
		values: [2]uint16{0x12, 0x34},
	})

	copied, ok := original.Copy().(InstructionArgument)
	assert.True(t, ok)
	assert.Equal(t, original.Value, copied.Value)
}

func TestInstructionArgument_CopyMutableValue(t *testing.T) {
	original := NewInstructionArgument(mutableInstructionArgument{values: []int{1, 2}})

	copied := original.Copy().(InstructionArgument)
	copiedValue := copied.Value.(mutableInstructionArgument)
	copiedValue.values[0] = 9
	originalValue := original.Value.(mutableInstructionArgument)

	assert.Equal(t, []int{1, 2}, originalValue.values)
	assert.Equal(t, []int{9, 2}, copiedValue.values)
}

func TestInstructionArgument_CopyRejectsUnsupportedMutableValues(t *testing.T) {
	values := []any{
		[]int{1},
		map[string]int{"a": 1},
		new(int),
		unsupportedMutableInstructionArgument{values: []int{1}},
	}

	for _, value := range values {
		t.Run(reflect.TypeOf(value).String(), func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered == nil {
					t.Fatal("expected copy to reject a mutable value without InstructionArgumentCopier")
				}
			}()
			NewInstructionArgument(value).Copy()
		})
	}
}

func TestInstructionArgument_CopyNilValue(t *testing.T) {
	copied := NewInstructionArgument(nil).Copy().(InstructionArgument)
	assert.Nil(t, copied.Value)

	var pointer *int
	copied = NewInstructionArgument(pointer).Copy().(InstructionArgument)
	assert.Nil(t, copied.Value)
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

func TestInstructionArguments_CopyPreservesNilNodes(t *testing.T) {
	original := NewInstructionArguments(nil, NewNumber(1))

	copied := original.Copy().(InstructionArguments)
	assert.Len(t, copied.Values, 2)
	assert.Nil(t, copied.Values[0])
	assert.Equal(t, uint64(1), copied.Values[1].(Number).Value)
}

func TestRegisterValue_Copy(t *testing.T) {
	original := NewRegisterValue(3, NewLabel("target"))

	copied, ok := original.Copy().(RegisterValue)
	assert.True(t, ok)
	assert.Equal(t, byte(3), copied.Register)

	label, ok := copied.Value.(Label)
	assert.True(t, ok)
	assert.Equal(t, "target", label.Name)
}

func TestRegisterRegisterValue_Copy(t *testing.T) {
	original := NewRegisterRegisterValue(1, 2, NewNumber(0x42))

	copied, ok := original.Copy().(RegisterRegisterValue)
	assert.True(t, ok)
	assert.Equal(t, byte(1), copied.Register1)
	assert.Equal(t, byte(2), copied.Register2)

	number, ok := copied.Value.(Number)
	assert.True(t, ok)
	assert.Equal(t, uint64(0x42), number.Value)
}
