package parser

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu68000"
	"github.com/retroenv/retrogolib/assert"
)

func TestResolvedInstruction_CopyInstructionArgument(t *testing.T) {
	original := ResolvedInstruction{
		Instruction: cpu68000.Instructions[cpu68000.MOVEName],
		SrcEA: &EffectiveAddress{
			Mode:  cpu68000.ImmediateMode,
			Value: ast.NewExpression(token.Token{Type: token.Identifier, Value: "source"}),
		},
		DstEA: &EffectiveAddress{Mode: cpu68000.DataRegDirectMode},
	}

	copied := original.CopyInstructionArgument().(ResolvedInstruction)
	copied.SrcEA.Mode = cpu68000.AbsLongMode
	copiedExpression := copied.SrcEA.Value.(ast.Expression)
	copiedExpression.Value.Tokens()[0].Value = "changed"
	copied.DstEA.Register = 3

	assert.Equal(t, cpu68000.ImmediateMode, original.SrcEA.Mode)
	assert.Equal(t, "source", original.SrcEA.Value.(ast.Expression).Value.Tokens()[0].Value)
	assert.Equal(t, uint8(0), original.DstEA.Register)
}

func TestResolvedInstruction_InstructionReferences(t *testing.T) {
	resolved := ResolvedInstruction{
		SrcEA: Immediate(ast.NewExpression(
			token.Token{Type: token.Identifier, Value: "source"},
			token.Token{Type: token.Plus},
			token.Token{Type: token.Number, Value: "2"},
		)),
		DstEA: Absolute(true, ast.NewLabel("destination")),
	}

	references := resolved.InstructionReferences()
	assert.Len(t, references, 2)
	symbol, addend, ok := ast.ParseSymbolReference(references[0].Value.(ast.Expression).Value)
	assert.True(t, ok)
	assert.Equal(t, "source", symbol)
	assert.Equal(t, int64(2), addend)
	assert.Equal(t, "destination", ast.SymbolName(references[1].Value))
	assert.Equal(t, ast.FullAddress, references[1].ReferenceType)

	references[0].Value.(ast.Expression).Value.Tokens()[0].Value = "changed"
	assert.Equal(t, "source", resolved.SrcEA.Value.(ast.Expression).Value.Tokens()[0].Value)
}
