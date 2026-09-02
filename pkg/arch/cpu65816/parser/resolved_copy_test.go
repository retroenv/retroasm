package parser

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu65816"
	"github.com/retroenv/retrogolib/assert"
)

func TestResolvedInstruction_CopyInstructionArgument(t *testing.T) {
	t.Parallel()

	original := ResolvedInstruction{
		Instruction: cpu65816.LdaInst,
		Addressing:  cpu65816.AbsoluteAddressing,
		Operands: Operands{MemoryOperand(
			OperandAddress,
			AddressAbsolute,
			ast.NewExpression(token.Token{Type: token.Identifier, Value: "source"}),
		)},
		State: DefaultState(),
	}

	copied := original.CopyInstructionArgument().(ResolvedInstruction)
	copiedExpression := copied.Operands[0].Value.(ast.Expression)
	copiedExpression.Value.Tokens()[0].Value = "changed"

	assert.Equal(t, "source", original.Operands[0].Value.(ast.Expression).Value.Tokens()[0].Value)
}
