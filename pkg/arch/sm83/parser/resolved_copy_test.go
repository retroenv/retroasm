package parser

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpusm83 "github.com/retroenv/retrogolib/arch/cpu/sm83"
	"github.com/retroenv/retrogolib/assert"
)

func TestResolvedInstruction_CopyInstructionArgument(t *testing.T) {
	original := ResolvedInstruction{
		Instruction:    cpusm83.LdImm8,
		RegisterParams: []cpusm83.RegisterParam{cpusm83.RegA},
		OperandValues: []ast.Node{
			ast.NewExpression(token.Token{Type: token.Identifier, Value: "source"}),
			nil,
		},
	}

	copied := original.CopyInstructionArgument().(ResolvedInstruction)
	copied.RegisterParams[0] = cpusm83.RegB
	copiedExpression := copied.OperandValues[0].(ast.Expression)
	copiedExpression.Value.Tokens()[0].Value = "changed"

	assert.Equal(t, cpusm83.RegA, original.RegisterParams[0])
	assert.Equal(t, "source", original.OperandValues[0].(ast.Expression).Value.Tokens()[0].Value)
	assert.Nil(t, copied.OperandValues[1])
}
