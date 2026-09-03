package parser

import (
	"fmt"
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

func TestResolvedInstruction_InstructionReferences(t *testing.T) {
	t.Parallel()

	resolved := ResolvedInstruction{Operands: Operands{
		{Kind: OperandImmediate, Value: ast.NewNumber(1)},
		{
			Kind:      OperandAddress,
			Value:     ast.NewLabel("target"),
			Modifiers: []ast.Modifier{{Operator: ast.NewOperator("+"), Value: "2"}},
		},
	}}

	references := resolved.InstructionReferences()
	assert.Len(t, references, 1)
	assert.Equal(t, "target", ast.SymbolName(references[0].Value))
	assert.Equal(t, "2", references[0].Modifiers[0].Value)
	assert.Equal(t, ast.FullAddress, references[0].ReferenceType)

	references[0].Modifiers[0].Value = "changed"
	assert.Equal(t, "2", resolved.Operands[1].Modifiers[0].Value)
}

func TestResolvedInstruction_InstructionFormKey(t *testing.T) {
	t.Parallel()

	resolved := ResolvedInstruction{
		Addressing: cpu65816.AbsoluteAddressing,
		Operands: Operands{{
			Kind: OperandAddress, Size: AddressLong, Value: ast.NewLabel("target"),
			Modifiers: []ast.Modifier{{Operator: ast.NewOperator("+"), Value: "2"}},
		}},
		State: State{
			AccumulatorWidth: WidthWord,
			IndexWidth:       WidthByte,
			Carry:            StatusSet,
			Emulation:        StatusClear,
		},
	}
	want := fmt.Sprintf(
		"addressing=%d;state=%d/%d/%d/%d;operands=%d/%d/+/ast.Label",
		cpu65816.AbsoluteAddressing,
		WidthWord,
		WidthByte,
		StatusSet,
		StatusClear,
		OperandAddress,
		AddressLong,
	)

	assert.Equal(t, want, resolved.InstructionFormKey())
	resolved.Operands[0].Value = ast.NewLabel("other")
	resolved.Operands[0].Modifiers[0].Value = "9"
	assert.Equal(t, want, resolved.InstructionFormKey())
	resolved.Operands[0].Value = ast.NewNumber(99)
	assert.NotEqual(t, want, resolved.InstructionFormKey())
}

func TestResolvedInstruction_InstructionStateTransitionForm(t *testing.T) {
	t.Parallel()

	rep := ResolvedInstruction{
		Instruction: cpu65816.RepInst,
		Operands:    Operands{ImmediateOperand(ast.NewNumber(0x30))},
		State:       DefaultState(),
	}
	assertInstructionStateTransition(t, rep, "rep:1/1/0/1->2/2/0/1", true)

	sep := ResolvedInstruction{
		Instruction: cpu65816.SepInst,
		Operands:    Operands{ImmediateOperand(ast.NewNumber(0x30))},
		State: State{
			AccumulatorWidth: WidthWord,
			IndexWidth:       WidthWord,
			Emulation:        StatusClear,
		},
	}
	assertInstructionStateTransition(t, sep, "sep:2/2/0/1->1/1/0/1", true)

	xce := ResolvedInstruction{
		Instruction: cpu65816.XceInst,
		State: State{
			AccumulatorWidth: WidthByte,
			IndexWidth:       WidthByte,
			Carry:            StatusSet,
			Emulation:        StatusClear,
		},
	}
	assertInstructionStateTransition(t, xce, "xce:1/1/2/1->1/1/1/2", true)

	lda := ResolvedInstruction{
		Instruction: cpu65816.LdaInst,
		Operands:    Operands{ImmediateOperand(ast.NewNumber(1))},
		State:       DefaultState(),
	}
	assertInstructionStateTransition(t, lda, "", false)
}

func assertInstructionStateTransition(t *testing.T, resolved ResolvedInstruction, want string, transition bool) {
	t.Helper()

	form, ok := resolved.InstructionStateTransitionForm()

	assert.Equal(t, transition, ok)
	assert.Equal(t, want, form)
}
