package ast

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retrogolib/arch"
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

func TestNodeCopiesOwnInlineComments(t *testing.T) {
	tests := []struct {
		name string
		node Node
	}{
		{name: "instruction", node: NewInstruction("nop", 0, nil, nil)},
		{name: "label", node: NewLabel("entry")},
		{name: "number", node: NewNumber(1)},
		{name: "identifier", node: NewIdentifier("value")},
		{name: "expression", node: NewExpression(token.Token{Type: token.Number, Value: "1"})},
		{name: "operator", node: NewOperator("+")},
		{name: "register value", node: NewRegisterValue(1, NewNumber(2))},
		{name: "register pair value", node: NewRegisterRegisterValue(1, 2, NewNumber(3))},
		{name: "instruction argument", node: NewInstructionArgument(1)},
		{name: "instruction arguments", node: NewInstructionArguments(NewNumber(1))},
		{name: "data", node: NewData(DataType, 1)},
		{name: "alias", node: NewAlias("value")},
		{name: "bank", node: NewBank(1)},
		{name: "base", node: NewBase(nil)},
		{name: "offset counter", node: NewOffsetCounter(1)},
		{name: "segment", node: NewSegment("code")},
		{name: "if", node: NewIf(nil)},
		{name: "ifdef", node: NewIfdef("value")},
		{name: "ifndef", node: NewIfndef("value")},
		{name: "else", node: NewElse()},
		{name: "else if", node: NewElseIf(nil)},
		{name: "endif", node: NewEndif()},
		{name: "configuration", node: Configuration{node: &node{}, Item: ConfigMapper, Expression: expression.New()}},
		{name: "repeat", node: NewRept(nil)},
		{name: "repeat end", node: NewEndr()},
		{name: "scope", node: NewScope("local")},
		{name: "scope end", node: NewScopeEnd()},
		{name: "enum", node: NewEnum(nil)},
		{name: "enum end", node: NewEnumEnd()},
		{name: "function", node: NewFunction("main")},
		{name: "function end", node: NewFunctionEnd()},
		{name: "include", node: NewInclude("data.bin", true, 0, 1)},
		{name: "macro", node: NewMacro("load")},
		{name: "error", node: NewError("failure")},
		{name: "variable", node: NewVariable("value", 1)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.node.SetComment("original")
			copied := test.node.Copy()
			copied.SetComment("copy")

			assert.Equal(t, "original", InlineComment(test.node))
			assert.Equal(t, "copy", InlineComment(copied))
		})
	}

	assert.Empty(t, InlineComment(&Comment{Message: "standalone"}))
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

func TestWithInstructionOpcodeID(t *testing.T) {
	identity := NewOpcodeID(arch.Z80, 7)
	original := NewInstruction("ld", 0, nil, nil)

	updated := WithInstructionOpcodeID(original, identity)
	instruction, ok := InstructionFromNode(updated)
	assert.True(t, ok)
	assert.Equal(t, identity, instruction.OpcodeID)
	assert.Equal(t, OpcodeID{}, original.OpcodeID)
	assert.True(t, identity.ValidFor(arch.Z80))
	assert.False(t, identity.ValidFor(arch.SM83))
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

	t.Run("data with value expressions", func(t *testing.T) {
		original := NewData(AddressType, 2)
		original.Values = []*expression.Expression{
			expression.New(token.Token{Type: token.Identifier, Value: "target"}),
			expression.New(token.Token{Type: token.Number, Value: "1"}),
		}
		original.ReferenceType = FullAddress
		original.Fill = true

		copied, ok := original.Copy().(Data)
		original.Values[0].AddTokens(token.Token{Type: token.Plus})

		assert.True(t, ok)
		assert.Equal(t, AddressType, copied.Type)
		assert.Equal(t, 2, copied.Width)
		assert.Equal(t, FullAddress, copied.ReferenceType)
		assert.True(t, copied.Fill)
		assert.Len(t, copied.Values, 2)
		assert.Len(t, copied.Values[0].Tokens(), 1)
		assert.NotNil(t, copied.Size)
	})
}

func TestData_Validate(t *testing.T) {
	newValue := func() *expression.Expression {
		return expression.New(token.Token{Type: token.Number, Value: "1"})
	}

	validData := NewData(DataType, 1)
	validData.Values = []*expression.Expression{newValue()}
	assert.NoError(t, validData.Validate())

	validFill := NewData(DataType, 2)
	validFill.Fill = true
	validFill.Size = newValue()
	assert.NoError(t, validFill.Validate())

	tests := []struct {
		name string
		data Data
	}{
		{name: "invalid type", data: NewData(InvalidDataType, 1)},
		{name: "invalid width", data: NewData(DataType, 0)},
		{name: "nil size", data: Data{node: &node{}, Type: DataType, Width: 1, Values: []*expression.Expression{newValue()}}},
		{name: "missing values", data: NewData(DataType, 1)},
		{name: "nil value", data: Data{node: &node{}, Type: DataType, Width: 1, Size: expression.New(), Values: []*expression.Expression{nil}}},
		{name: "empty value", data: Data{node: &node{}, Type: DataType, Width: 1, Size: expression.New(), Values: []*expression.Expression{expression.New()}}},
		{name: "data with reference type", data: Data{node: &node{}, Type: DataType, Width: 1, ReferenceType: FullAddress, Size: expression.New(), Values: []*expression.Expression{newValue()}}},
		{name: "address with invalid reference", data: Data{node: &node{}, Type: AddressType, Width: 2, Size: expression.New(), Values: []*expression.Expression{newValue()}}},
		{name: "address with expression", data: Data{node: &node{}, Type: AddressType, Width: 2, ReferenceType: FullAddress, Size: expression.New(), Values: []*expression.Expression{expression.New(token.Token{Type: token.Number, Value: "1"}, token.Token{Type: token.Plus, Value: "+"}, token.Token{Type: token.Number, Value: "2"})}}},
		{name: "address fill", data: Data{node: &node{}, Type: AddressType, Width: 2, ReferenceType: FullAddress, Fill: true, Size: newValue(), Values: []*expression.Expression{newValue()}}},
		{name: "fill without size", data: Data{node: &node{}, Type: DataType, Width: 1, Fill: true, Size: expression.New()}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.ErrorIs(t, test.data.Validate(), ErrInvalidData)
		})
	}

	address := NewData(AddressType, 2)
	address.ReferenceType = FullAddress
	address.Values = []*expression.Expression{newValue()}
	for _, candidate := range []Node{address, &address} {
		data, ok := DataFromNode(candidate)
		assert.True(t, ok)
		assert.Equal(t, AddressType, data.Type)
	}
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
