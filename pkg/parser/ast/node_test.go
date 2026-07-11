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

func TestInstructionFromNode(t *testing.T) {
	instr := NewInstruction("lda", 1, NewNumber(42), nil)

	for _, node := range []Node{instr, &instr} {
		got, ok := InstructionFromNode(node)
		assert.True(t, ok)
		assert.Equal(t, instr, got)
	}

	var nilInstr *Instruction
	_, ok := InstructionFromNode(nilInstr)
	assert.False(t, ok)
}

func TestIsInstruction(t *testing.T) {
	instr := NewInstruction("nop", 0, nil, nil)
	assert.True(t, IsInstruction(instr))
	assert.True(t, IsInstruction(&instr))

	var nilInstr *Instruction
	assert.False(t, IsInstruction(nilInstr))
	assert.False(t, IsInstruction(NewNumber(1)))
}

func TestIsLabel(t *testing.T) {
	label := NewLabel("loop")
	assert.True(t, IsLabel(label))
	assert.True(t, IsLabel(&label))

	var nilLabel *Label
	assert.False(t, IsLabel(nilLabel))
	assert.False(t, IsLabel(NewNumber(1)))
}

func TestLabelIndices(t *testing.T) {
	label := NewLabel("start")
	nodes := []Node{
		&label,
		NewInstruction("nop", 0, nil, nil),
		NewLabel("done"),
	}

	indices := LabelIndices(nodes)
	assert.Len(t, indices, 2)
	assert.Equal(t, 0, indices["start"])
	assert.Equal(t, 2, indices["done"])
}

func TestFillLabelIndices(t *testing.T) {
	indices := map[string]int{"stale": 7}
	FillLabelIndices([]Node{NewLabel("only")}, indices)

	assert.Len(t, indices, 1)
	assert.Equal(t, 0, indices["only"])
}

func TestIdentifierName(t *testing.T) {
	identifier := NewIdentifier("target")

	for _, node := range []Node{identifier, &identifier} {
		got, ok := IdentifierName(node)
		assert.True(t, ok)
		assert.Equal(t, identifier.Name, got)
	}

	var nilIdentifier *Identifier
	_, ok := IdentifierName(nilIdentifier)
	assert.False(t, ok)
}

func TestLabelName(t *testing.T) {
	label := NewLabel("loop")

	for _, node := range []Node{label, &label} {
		got, ok := LabelName(node)
		assert.True(t, ok)
		assert.Equal(t, label.Name, got)
	}

	var nilLabel *Label
	_, ok := LabelName(nilLabel)
	assert.False(t, ok)
}

func TestNumberValue(t *testing.T) {
	number := NewNumber(42)

	for _, node := range []Node{number, &number} {
		got, ok := NumberValue(node)
		assert.True(t, ok)
		assert.Equal(t, number.Value, got)
	}

	var nilNumber *Number
	_, ok := NumberValue(nilNumber)
	assert.False(t, ok)
}

func TestSymbolName(t *testing.T) {
	label := NewLabel("loop")
	identifier := NewIdentifier("target")
	tests := []struct {
		node Node
		want string
	}{
		{node: label, want: label.Name},
		{node: &label, want: label.Name},
		{node: identifier, want: identifier.Name},
		{node: &identifier, want: identifier.Name},
	}
	for _, test := range tests {
		assert.Equal(t, test.want, SymbolName(test.node))
	}

	var nilIdentifier *Identifier
	assert.Equal(t, "", SymbolName(nilIdentifier))
}

func TestSameOperand(t *testing.T) {
	number := NewNumber(42)
	identifier := NewIdentifier("target")
	label := NewLabel("target")

	assert.True(t, SameOperand(number, &number))
	assert.True(t, SameOperand(identifier, &label))
	assert.True(t, SameOperand(nil, nil))
	assert.False(t, SameOperand(number, identifier))
	assert.False(t, SameOperand(identifier, NewIdentifier("other")))
}

func TestInstruction_Copy(t *testing.T) {
	original := NewInstruction("lda", 1, NewNumber(42), nil)
	original.SetComment("load accumulator")

	copied, ok := original.Copy().(Instruction)
	assert.True(t, ok)
	assert.Equal(t, "lda", copied.Name)
	assert.Equal(t, 1, copied.Addressing)
}

func TestInstruction_ArgumentSymbolName(t *testing.T) {
	labelInstr := NewInstruction("beq", 1, NewLabel("done"), nil)
	assert.Equal(t, "done", labelInstr.ArgumentSymbolName())

	identifierInstr := NewInstruction("jmp", 1, NewIdentifier("main"), nil)
	assert.Equal(t, "main", identifierInstr.ArgumentSymbolName())

	numberInstr := NewInstruction("lda", 1, NewNumber(42), nil)
	assert.Equal(t, "", numberInstr.ArgumentSymbolName())
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

func TestScope_Copy(t *testing.T) {
	original := NewScope("inner")
	original.SetComment("nested scope")

	copied, ok := original.Copy().(Scope)
	assert.True(t, ok)
	assert.Equal(t, "inner", copied.Name)
}

func TestScopeEnd_Copy(t *testing.T) {
	original := NewScopeEnd()
	original.SetComment("end nested scope")

	_, ok := original.Copy().(ScopeEnd)
	assert.True(t, ok)
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
