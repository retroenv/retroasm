package parser

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpusm83 "github.com/retroenv/retrogolib/arch/cpu/sm83"
	"github.com/retroenv/retrogolib/assert"
)

func TestResolvedInstruction_CopyInstructionArgument(t *testing.T) {
	original := ResolvedInstruction{
		Instruction:    cpusm83.LdImm8,
		RegisterParams: []cpusm83.RegisterParam{cpusm83.RegA},
		OperandValues:  []ast.Node{ast.NewNumber(1)},
	}

	copied := original.CopyInstructionArgument().(ResolvedInstruction)
	copied.RegisterParams[0] = cpusm83.RegB
	copied.OperandValues[0] = ast.NewNumber(2)

	assert.Equal(t, cpusm83.RegA, original.RegisterParams[0])
	assert.Equal(t, uint64(1), original.OperandValues[0].(ast.Number).Value)
}
