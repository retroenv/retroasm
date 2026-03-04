package ast

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retrogolib/assert"
)

func TestNode_SetComment(t *testing.T) {
	t.Run("set comment on base node", func(t *testing.T) {
		n := &node{}
		n.SetComment("test comment")
		assert.Equal(t, "test comment", n.comment.Message)
	})

	t.Run("set comment on instruction", func(t *testing.T) {
		inst := NewInstruction("nop", 0, NewNumber(42), nil)
		inst.SetComment("instruction comment")
		assert.NotNil(t, inst.Copy())
	})

	t.Run("set comment on label", func(t *testing.T) {
		label := NewLabel("main")
		label.SetComment("main function entry")

		copied, ok := label.Copy().(Label)
		assert.True(t, ok)
		assert.Equal(t, "main", copied.Name)
	})
}

func TestInstruction_Copy(t *testing.T) {
	original := NewInstruction("lda", 1, NewNumber(42), nil)
	original.SetComment("load accumulator")

	copied, ok := original.Copy().(Instruction)
	assert.True(t, ok)
	assert.Equal(t, "lda", copied.Name)
	assert.Equal(t, 1, copied.Addressing)
}

func TestLabel_Copy(t *testing.T) {
	original := NewLabel("loop")
	original.SetComment("main loop")

	copied, ok := original.Copy().(Label)
	assert.True(t, ok)
	assert.Equal(t, "loop", copied.Name)
}

func TestNumber_Copy(t *testing.T) {
	original := NewNumber(255)

	copied, ok := original.Copy().(Number)
	assert.True(t, ok)
	assert.Equal(t, uint64(255), copied.Value)
}

func TestExpression_Copy(t *testing.T) {
	original := NewExpression(
		token.Token{Type: token.Identifier, Value: "target"},
		token.Token{Type: token.Plus},
		token.Token{Type: token.Number, Value: "1"},
	)

	copied, ok := original.Copy().(Expression)
	assert.True(t, ok)
	assert.NotNil(t, copied.Value)
	assert.Len(t, copied.Value.Tokens(), 3)
}

func TestData_Copy(t *testing.T) {
	t.Run("data with nil values", func(t *testing.T) {
		original := NewData(DataType, 1)

		copied, ok := original.Copy().(Data)
		assert.True(t, ok)
		assert.Equal(t, DataType, copied.Type)
		assert.Equal(t, 1, copied.Width)
		assert.NotNil(t, copied.Size)
		assert.Nil(t, copied.Values)
	})

	t.Run("data with values expression", func(t *testing.T) {
		original := NewData(AddressType, 2)
		original.Values = expression.New()
		original.ReferenceType = FullAddress
		original.Fill = true

		copied, ok := original.Copy().(Data)
		assert.True(t, ok)
		assert.Equal(t, AddressType, copied.Type)
		assert.Equal(t, 2, copied.Width)
		assert.Equal(t, FullAddress, copied.ReferenceType)
		assert.True(t, copied.Fill)
		assert.NotNil(t, copied.Values)
		assert.NotNil(t, copied.Size)
	})
}

func TestAlias_Copy(t *testing.T) {
	original := NewAlias("SCREEN")

	copied, ok := original.Copy().(Alias)
	assert.True(t, ok)
	assert.Equal(t, "SCREEN", copied.Name)
}

func TestOffsetCounter_Copy(t *testing.T) {
	original := NewOffsetCounter(42)
	assert.Equal(t, uint64(42), original.Number)

	copyOC, ok := original.Copy().(OffsetCounter)
	assert.True(t, ok)
	assert.Equal(t, uint64(42), copyOC.Number)
}

func TestAST_EdgeCases(t *testing.T) {
	t.Run("empty string values", func(t *testing.T) {
		label := NewLabel("")
		assert.Equal(t, "", label.Name)

		alias := NewAlias("")
		assert.Equal(t, "", alias.Name)

		ident := NewIdentifier("")
		assert.Equal(t, "", ident.Name)
	})

	t.Run("zero values", func(t *testing.T) {
		num := NewNumber(0)
		assert.Equal(t, uint64(0), num.Value)

		bank := NewBank(0)
		assert.Equal(t, 0, bank.Number)

		variable := NewVariable("var", 0)
		assert.Equal(t, 0, variable.Size)
	})

	t.Run("negative values where applicable", func(t *testing.T) {
		bank := NewBank(-1)
		assert.Equal(t, -1, bank.Number)

		variable := NewVariable("var", -5)
		assert.Equal(t, -5, variable.Size)
	})
}
