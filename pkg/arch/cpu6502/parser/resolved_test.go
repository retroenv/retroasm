package parser

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu6502"
	"github.com/retroenv/retrogolib/assert"
)

func TestResolveInstruction_ReturnsIsolatedTypedProjection(t *testing.T) {
	t.Parallel()

	expression := ast.NewExpression(
		token.Token{Type: token.LeftParentheses, Value: "("},
		token.Token{Type: token.Identifier, Value: "source"},
		token.Token{Type: token.RightParentheses, Value: ")"},
	)
	instruction := ast.NewInstruction(
		cpu6502.LdaName,
		int(cpu6502.ImmediateAddressing),
		expression,
		nil,
	)
	resolved, err := ResolveInstruction(instruction, cpu6502.LdaInst)
	assert.NoError(t, err)
	resolvedExpression := resolved.Operands[0].Value.(ast.Expression)
	resolvedExpression.Value.Tokens()[1].Value = "changed"

	assert.Equal(t, "source", instruction.Argument.(ast.Expression).Value.Tokens()[1].Value)
	assert.Equal(t, "changed", resolvedExpression.Value.Tokens()[1].Value)
}
