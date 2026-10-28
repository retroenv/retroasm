package x86_test

import (
	"testing"

	asmx86 "github.com/retroenv/retroasm/pkg/arch/x86"
	"github.com/retroenv/retroasm/pkg/arch/x86/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	isa "github.com/retroenv/retrogolib/arch/cpu/x86"
	"github.com/retroenv/retrogolib/assert"
)

func TestTypedRegisterOperandsRetainSymbolContracts(t *testing.T) {
	operands := []ast.Node{ast.NewInstructionArgument(isa.RegAX), ast.NewLabel("source")}
	instruction, err := asmx86.NewInstruction(isa.MovName, operands...)
	assert.NoError(t, err)
	assert.Equal(t, isa.RegAX, operands[0].(ast.InstructionArgument).Value)
	argument := instruction.Argument.(ast.InstructionArguments).Values[0].(ast.InstructionArgument)
	assert.Equal(t, "x86-register/ax", ast.InstructionArgumentForm(argument))
	references, valid := argument.Value.(ast.InstructionReferenceProvider)
	assert.True(t, valid)
	assert.Empty(t, references.InstructionReferences())
	rewritten, err := ast.RewriteNodeSymbols(instruction, renameX86Source)
	assert.NoError(t, err)
	text, err := parser.FormatInstruction(rewritten.(ast.Instruction))
	assert.NoError(t, err)
	assert.Equal(t, "mov ax, renamed_source", text)
	text, err = parser.FormatInstruction(instruction)
	assert.NoError(t, err)
	assert.Equal(t, "mov ax, source", text)
}

func renameX86Source(name string) string {
	return "renamed_" + name
}
