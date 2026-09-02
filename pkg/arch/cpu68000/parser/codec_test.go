package parser

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu68000"
	"github.com/retroenv/retrogolib/assert"
)

func TestFormatInstructionUsesCPU68000FlowAndAddressSyntax(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		mnemonic string
		operands Operands
		want     string
	}{
		{
			name:     "branch label is not an absolute address",
			mnemonic: "BNE",
			operands: UnaryOperands(cpu68000.SizeWord, Absolute(true, ast.NewLabel("target"))),
			want:     "  BNE target",
		},
		{
			name:     "lea has no data-size suffix",
			mnemonic: "LEA",
			operands: BinaryOperands(
				cpu68000.SizeLong,
				Displacement(7, ast.NewNumber(4)),
				AddressRegister(0),
			),
			want: "  LEA 4(SP),A0",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			metadata, ok := cpu68000.Instructions[func() string {
				base, _, conditional := ParseConditionCode(test.mnemonic)
				if conditional {
					return base
				}
				return test.mnemonic
			}()]
			assert.True(t, ok)
			instruction, err := BuildInstruction(test.mnemonic, metadata, test.operands)
			assert.NoError(t, err)
			formatted, err := FormatInstructionWithOptions(instruction, FormatOptions{
				Indent:               "  ",
				Uppercase:            true,
				DecimalValues:        true,
				StackPointerAlias:    true,
				OmitWordBranchSuffix: true,
			})
			assert.NoError(t, err)
			assert.Equal(t, test.want, formatted)
		})
	}
}
