package parser

import (
	"fmt"
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
		Operands: []Operand{
			ValueOperand(ast.NewExpression(token.Token{Type: token.Identifier, Value: "operand"})),
		},
		OperandValues: []ast.Node{
			ast.NewExpression(token.Token{Type: token.Identifier, Value: "source"}),
			nil,
		},
	}

	copied := original.CopyInstructionArgument().(ResolvedInstruction)
	copied.RegisterParams[0] = cpuz80.RegB
	copiedExpression := copied.OperandValues[0].(ast.Expression)
	copiedExpression.Value.Tokens()[0].Value = "changed"
	copiedOperandExpression := copied.Operands[0].Value.(ast.Expression)
	copiedOperandExpression.Value.Tokens()[0].Value = "changed-operand"

	assert.Equal(t, cpuz80.RegA, original.RegisterParams[0])
	assert.Equal(t, "source", original.OperandValues[0].(ast.Expression).Value.Tokens()[0].Value)
	assert.Equal(t, "operand", original.Operands[0].Value.(ast.Expression).Value.Tokens()[0].Value)
	assert.Nil(t, copied.OperandValues[1])
}

func TestResolvedInstruction_InstructionReferences(t *testing.T) {
	resolved := ResolvedInstruction{OperandValues: []ast.Node{
		ast.NewNumber(1),
		ast.NewLabel("direct"),
		ast.NewExpression(
			token.Token{Type: token.Identifier, Value: "target"},
			token.Token{Type: token.Plus},
			token.Token{Type: token.Number, Value: "2"},
		),
	}}

	references := resolved.InstructionReferences()
	assert.Len(t, references, 2)
	assert.Equal(t, "direct", ast.SymbolName(references[0].Value))
	symbol, addend, ok := ast.ParseSymbolReference(references[1].Value.(ast.Expression).Value)
	assert.True(t, ok)
	assert.Equal(t, "target", symbol)
	assert.Equal(t, int64(2), addend)
	assert.Equal(t, ast.FullAddress, references[1].ReferenceType)

	references[1].Value.(ast.Expression).Value.Tokens()[0].Value = "changed"
	assert.Equal(t, "target", resolved.OperandValues[2].(ast.Expression).Value.Tokens()[0].Value)
}

func TestResolvedInstruction_InstructionFormKey(t *testing.T) {
	t.Parallel()

	resolved := ResolvedInstruction{
		Addressing:     cpuz80.RegisterIndirectAddressing,
		RegisterParams: []cpuz80.RegisterParam{cpuz80.RegA, cpuz80.RegIXIndirect},
		Operands: []Operand{
			RegisterOperand(cpuz80.RegA),
			IndexedOperand(cpuz80.RegIX, ast.NewLabel("target")),
		},
	}
	want := fmt.Sprintf(
		"addressing=%d;registers=%d,%d;operands=%d/%d/none,%d/%d/ast.Label",
		cpuz80.RegisterIndirectAddressing,
		cpuz80.RegA,
		cpuz80.RegIXIndirect,
		OperandRegister,
		cpuz80.RegA,
		OperandIndexed,
		cpuz80.RegIX,
	)

	assert.Equal(t, want, resolved.InstructionFormKey())
	resolved.Operands[1].Value = ast.NewLabel("other")
	assert.Equal(t, want, resolved.InstructionFormKey())
}
