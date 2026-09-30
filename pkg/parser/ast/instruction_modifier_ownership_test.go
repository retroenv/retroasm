package ast

import (
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

func TestInstructionCopyKeepsModifierOperatorsIndependent(t *testing.T) {
	operator := NewOperator("+")
	operator.SetComment("source operator")
	original := NewInstruction("lda", 1, NewNumber(1), []Modifier{{
		Operator: operator,
		Value:    "1",
	}})
	copied := original.Copy().(Instruction)
	copied.Modifier[0].Operator.SetComment("copy operator")
	assert.Equal(t, "source operator", InlineComment(original.Modifier[0].Operator))
	assert.Equal(t, "copy operator", InlineComment(copied.Modifier[0].Operator))
}

func TestNodeEditModifierReadsDoNotChangeSource(t *testing.T) {
	operator := NewOperator("+")
	operator.SetComment("source operator")
	stream := NewStreamFromNodes(NewInstruction("lda", 1, NewNumber(1), []Modifier{{
		Operator: operator,
		Value:    "1",
	}}))
	view, err := stream.EditNodes()
	assert.NoError(t, err)
	read := view.At(0).(Instruction)
	read.Modifier[0].Operator.SetComment("caller operator")
	assert.Equal(t, "source operator", InlineComment(stream.At(0).Node.(Instruction).Modifier[0].Operator))
	assert.Equal(t, "source operator", InlineComment(view.At(0).(Instruction).Modifier[0].Operator))
	assert.NoError(t, view.Commit([]Node{read}))
	assert.Equal(t, "caller operator", InlineComment(stream.At(0).Node.(Instruction).Modifier[0].Operator))
	read.Modifier[0].Operator.SetComment("later caller")
	assert.Equal(t, "caller operator", InlineComment(stream.At(0).Node.(Instruction).Modifier[0].Operator))
}
