package parser

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu68000"
	"github.com/retroenv/retrogolib/assert"
)

func TestResolvedInstruction_CopyInstructionArgument(t *testing.T) {
	original := ResolvedInstruction{
		Instruction: cpu68000.Instructions[cpu68000.MOVEName],
		SrcEA: &EffectiveAddress{
			Mode:  cpu68000.ImmediateMode,
			Value: ast.NewNumber(1),
		},
		DstEA: &EffectiveAddress{Mode: cpu68000.DataRegDirectMode},
	}

	copied := original.CopyInstructionArgument().(ResolvedInstruction)
	copied.SrcEA.Mode = cpu68000.AbsLongMode
	copied.SrcEA.Value = ast.NewNumber(2)
	copied.DstEA.Register = 3

	assert.Equal(t, cpu68000.ImmediateMode, original.SrcEA.Mode)
	assert.Equal(t, uint64(1), original.SrcEA.Value.(ast.Number).Value)
	assert.Equal(t, uint8(0), original.DstEA.Register)
}
