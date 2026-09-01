package parser

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
	"github.com/retroenv/retrogolib/assert"
)

func TestResolvedInstruction_CopyInstructionArgument(t *testing.T) {
	original := ResolvedInstruction{
		Instruction:    cpuz80.LdImm8,
		RegisterParams: []cpuz80.RegisterParam{cpuz80.RegA},
		OperandValues: []ast.Node{
			ast.NewExpression(token.Token{Type: token.Identifier, Value: "source"}),
			nil,
		},
	}

	copied := original.CopyInstructionArgument().(ResolvedInstruction)
	copied.RegisterParams[0] = cpuz80.RegB
	copiedExpression := copied.OperandValues[0].(ast.Expression)
	copiedExpression.Value.Tokens()[0].Value = "changed"

	assert.Equal(t, cpuz80.RegA, original.RegisterParams[0])
	assert.Equal(t, "source", original.OperandValues[0].(ast.Expression).Value.Tokens()[0].Value)
	assert.Nil(t, copied.OperandValues[1])
}
