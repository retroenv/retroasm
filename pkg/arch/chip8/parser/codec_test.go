package parser

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/chip8"
	"github.com/retroenv/retrogolib/assert"
)

func TestFormatInstructionWithOptions_IndependentCaseAndSpacing(t *testing.T) {
	t.Parallel()

	instruction, err := BuildInstruction(
		chip8.LdName,
		chip8.LdInst,
		Operands{RegisterOperand(1), ByteOperand(ast.NewNumber(0x2a))},
	)
	assert.NoError(t, err)

	formatted, err := FormatInstructionWithOptions(instruction, FormatOptions{
		Indent:            "    ",
		UppercaseOperands: true,
		SpaceAfterComma:   true,
	})
	assert.NoError(t, err)
	assert.Equal(t, "    ld V1, 0x2A", formatted)
}
