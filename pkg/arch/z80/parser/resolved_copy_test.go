package parser

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
	"github.com/retroenv/retrogolib/assert"
)

func TestResolvedInstruction_CopyInstructionArgument(t *testing.T) {
	original := ResolvedInstruction{
		Instruction:    cpuz80.LdImm8,
		RegisterParams: []cpuz80.RegisterParam{cpuz80.RegA},
		OperandValues:  []ast.Node{ast.NewNumber(1)},
	}

	copied := original.CopyInstructionArgument().(ResolvedInstruction)
	copied.RegisterParams[0] = cpuz80.RegB
	copied.OperandValues[0] = ast.NewNumber(2)

	assert.Equal(t, cpuz80.RegA, original.RegisterParams[0])
	assert.Equal(t, uint64(1), original.OperandValues[0].(ast.Number).Value)
}
