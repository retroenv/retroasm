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
