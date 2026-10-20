package parser_test

import (
	"testing"

	asmcpu68000 "github.com/retroenv/retroasm/pkg/arch/cpu68000"
	cpu68000parser "github.com/retroenv/retroasm/pkg/arch/cpu68000/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu68000"
	"github.com/retroenv/retrogolib/assert"
)

func TestCPU68000DotSymbolTypedTextRoundTrip(t *testing.T) {
	for _, test := range []struct {
		mnemonic string
		operands cpu68000parser.Operands
	}{
		{
			mnemonic: "LEA",
			operands: cpu68000parser.BinaryOperands(cpu68000.SizeLong,
				cpu68000parser.Absolute(true, ast.NewLabel(".Ltable")), cpu68000parser.AddressRegister(0)),
		},
		{
			mnemonic: "MOVE",
			operands: cpu68000parser.BinaryOperands(cpu68000.SizeLong,
				cpu68000parser.Immediate(ast.NewLabel(".Ltable")), cpu68000parser.DataRegister(0)),
		},
		{
			mnemonic: "JMP",
			operands: cpu68000parser.UnaryOperands(cpu68000.SizeWord,
				cpu68000parser.Absolute(true, ast.NewLabel(".Ldone"))),
		},
	} {
		t.Run(test.mnemonic, func(t *testing.T) {
			built, err := cpu68000parser.BuildInstruction(test.mnemonic, cpu68000.Instructions[test.mnemonic], test.operands)
			assert.NoError(t, err)
			formatted, err := cpu68000parser.FormatInstruction(built)
			assert.NoError(t, err)
			parsed := parseInstruction(t, asmcpu68000.New().Arch, formatted)
			formattedAgain, err := cpu68000parser.FormatInstruction(parsed)
			assert.NoError(t, err)
			assert.Equal(t, formatted, formattedAgain)
		})
	}
}
