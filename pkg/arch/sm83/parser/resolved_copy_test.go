package parser

import (
	"fmt"
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
		Operands: []Operand{
			ValueOperand(ast.NewExpression(token.Token{Type: token.Identifier, Value: "operand"})),
		},
	}

	copied := original.CopyInstructionArgument().(ResolvedInstruction)
	copied.RegisterParams[0] = cpusm83.RegB
	copiedExpression := copied.OperandValues[0].(ast.Expression)
	copiedExpression.Value.Tokens()[0].Value = "changed"
	copiedOperandExpression := copied.Operands[0].Value.(ast.Expression)
	copiedOperandExpression.Value.Tokens()[0].Value = "changed operand"

	assert.Equal(t, cpusm83.RegA, original.RegisterParams[0])
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
			token.Token{Type: token.Minus},
			token.Token{Type: token.Number, Value: "2"},
		),
	}}

	references := resolved.InstructionReferences()
	assert.Len(t, references, 2)
	assert.Equal(t, "direct", ast.SymbolName(references[0].Value))
	symbol, addend, ok := ast.ParseSymbolReference(references[1].Value.(ast.Expression).Value)
	assert.True(t, ok)
	assert.Equal(t, "target", symbol)
	assert.Equal(t, int64(-2), addend)
	assert.Equal(t, ast.FullAddress, references[1].ReferenceType)

	references[1].Value.(ast.Expression).Value.Tokens()[0].Value = "changed"
	assert.Equal(t, "target", resolved.OperandValues[2].(ast.Expression).Value.Tokens()[0].Value)
}

func TestResolvedInstruction_InstructionFormKey(t *testing.T) {
	t.Parallel()

	resolved := ResolvedInstruction{
		Addressing:     cpusm83.RegisterIndirectAddressing,
		RegisterParams: []cpusm83.RegisterParam{cpusm83.RegA, cpusm83.RegHLIndirect},
		Operands: []Operand{
			RegisterOperand(cpusm83.RegA),
			HLIncrementOperand(),
		},
	}
	want := fmt.Sprintf(
		"addressing=%d;registers=%d,%d;operands=%d/%d/none,%d/0/none",
		cpusm83.RegisterIndirectAddressing,
		cpusm83.RegA,
		cpusm83.RegHLIndirect,
		OperandRegister,
		cpusm83.RegA,
		OperandHLIncrement,
	)

	assert.Equal(t, want, resolved.InstructionFormKey())
}
