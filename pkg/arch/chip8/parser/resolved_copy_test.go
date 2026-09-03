package parser

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/chip8"
	"github.com/retroenv/retrogolib/assert"
)

func TestResolvedInstruction_CopyInstructionArgument(t *testing.T) {
	t.Parallel()

	original := ResolvedInstruction{
		Instruction: chip8.LdInst,
		Addressing:  chip8.RegisterValueAddressing,
		Operands: Operands{
			RegisterOperand(1),
			ByteOperand(ast.NewExpression(token.Token{Type: token.Identifier, Value: "source"})),
		},
	}
	copied := original.CopyInstructionArgument().(ResolvedInstruction)
	expression := copied.Operands[1].Value.(ast.Expression)
	expression.Value.Tokens()[0].Value = "changed"

	assert.Equal(t, "source", original.Operands[1].Value.(ast.Expression).Value.Tokens()[0].Value)
	assert.Equal(t, "changed", expression.Value.Tokens()[0].Value)
}

func TestResolvedInstruction_InstructionReferences(t *testing.T) {
	t.Parallel()

	resolved := ResolvedInstruction{Operands: Operands{
		RegisterOperand(0),
		AddressOperand(ast.NewExpression(
			token.Token{Type: token.Identifier, Value: "target"},
			token.Token{Type: token.Plus},
			token.Token{Type: token.Number, Value: "2"},
		)),
		NibbleOperand(ast.NewNumber(1)),
	}}

	references := resolved.InstructionReferences()
	assert.Len(t, references, 1)
	symbol, addend, ok := ast.ParseSymbolReference(references[0].Value.(ast.Expression).Value)
	assert.True(t, ok)
	assert.Equal(t, "target", symbol)
	assert.Equal(t, int64(2), addend)
	assert.Equal(t, ast.FullAddress, references[0].ReferenceType)

	references[0].Value.(ast.Expression).Value.Tokens()[0].Value = "changed"
	assert.Equal(t, "target", resolved.Operands[1].Value.(ast.Expression).Value.Tokens()[0].Value)
}
