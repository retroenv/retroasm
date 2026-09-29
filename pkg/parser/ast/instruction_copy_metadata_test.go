package ast

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/assert"
)

func TestInstructionCopyKeepsMetadataIndependent(t *testing.T) {
	t.Parallel()
	identifier := NewIdentifier("value")
	identifier.Arguments = []token.Token{{Type: token.Number, Value: "1"}}
	for _, argument := range []Node{NewNumber(42), identifier, NewLabel("value"), nil} {
		original := NewInstruction("lda", 1, argument, []Modifier{{Value: "1"}})
		original.OpcodeID = NewOpcodeID(arch.Z80, 7)
		original.SetComment("instruction")
		original.setEntryHandle(&entryHandle{})
		if argument != nil {
			argument.SetComment("operand")
			argument.(interface{ setEntryHandle(*entryHandle) }).setEntryHandle(&entryHandle{})
		}
		copied := original.Copy().(Instruction)
		assert.Equal(t, original, copied)
		assert.Equal(t, "instruction", InlineComment(copied))
		copied.SetComment("changed instruction")
		copied.setEntryHandle(nil)
		copied.Modifier[0].Value = "2"
		assert.Equal(t, "instruction", InlineComment(original))
		assert.NotNil(t, original.entryHandle())
		assert.Equal(t, "1", original.Modifier[0].Value)
		if argument == nil {
			continue
		}
		assert.Equal(t, "operand", InlineComment(copied.Argument))
		copied.Argument.SetComment("changed operand")
		copied.Argument.(interface{ setEntryHandle(*entryHandle) }).setEntryHandle(nil)
		assert.Equal(t, "operand", InlineComment(original.Argument))
		assert.NotNil(t, original.Argument.(interface{ entryHandle() *entryHandle }).entryHandle())
		assert.Equal(t, "changed instruction", InlineComment(copied))
		if copiedIdentifier, ok := copied.Argument.(Identifier); ok {
			copiedIdentifier.Arguments[0].Value = "2"
			assert.Equal(t, "1", original.Argument.(Identifier).Arguments[0].Value)
		}
	}
}

func TestInstructionCopyAcceptsNilMetadata(t *testing.T) {
	t.Parallel()
	for _, argument := range []Node{Number{Value: 42}, Identifier{Name: "value"}} {
		original := Instruction{Argument: argument}
		copied := original.Copy().(Instruction)
		copied.SetComment("instruction")
		copied.Argument.SetComment("operand")
		assert.Empty(t, InlineComment(original))
		assert.Empty(t, InlineComment(original.Argument))
		assert.Equal(t, "instruction", InlineComment(copied))
		assert.Equal(t, "operand", InlineComment(copied.Argument))
	}
}
