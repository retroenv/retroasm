package codec_test

import (
	"strings"
	"testing"

	cpu68000parser "github.com/retroenv/retroasm/pkg/arch/cpu68000/parser"
	"github.com/retroenv/retroasm/pkg/codec"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu68000"
	"github.com/retroenv/retrogolib/assert"
)

func TestCPU68000StatusRegisterEncodings(t *testing.T) {
	for _, test := range []struct {
		line string
		want []byte
	}{
		{
			line: "move.w sr,d2",
			want: []byte{0x40, 0xc2},
		},
		{
			line: "move.w sr,(a1)",
			want: []byte{0x40, 0xd1},
		},
		{
			line: "move.w d2,ccr",
			want: []byte{0x44, 0xc2},
		},
		{
			line: "move.w (sp)+,ccr",
			want: []byte{0x44, 0xdf},
		},
		{
			line: "move.w #31,ccr",
			want: []byte{0x44, 0xfc, 0, 31},
		},
		{
			line: "andi.b #239,ccr",
			want: []byte{0x02, 0x3c, 0, 239},
		},
		{
			line: "ori.b #16,ccr",
			want: []byte{0, 0x3c, 0, 16},
		},
		{
			line: "eori.b #4,ccr",
			want: []byte{0x0a, 0x3c, 0, 4},
		},
	} {
		t.Run(test.line, func(t *testing.T) {
			c := newCPU68000Codec(t)
			parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(test.line))
			assert.NoError(t, err)
			formatted, err := c.FormatInstruction(parsed)
			assert.NoError(t, err)
			reparsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
			assert.NoError(t, err)
			for _, instruction := range []ast.Instruction{parsed, reparsed} {
				result, err := c.Assemble(t.Context(), []ast.Node{instruction})
				assert.NoError(t, err)
				assert.Equal(t, test.want, result.Binary)
			}
		})
	}
}

func TestCPU68000StatusOperandsBuildTypedInstructions(t *testing.T) {
	c := newCPU68000Codec(t)
	for _, test := range []struct {
		src, dst *cpu68000parser.EffectiveAddress
		want     []byte
	}{
		{
			src:  cpu68000parser.StatusRegister(),
			dst:  cpu68000parser.DataRegister(2),
			want: []byte{0x40, 0xc2},
		},
		{
			src:  cpu68000parser.DataRegister(2),
			dst:  cpu68000parser.ConditionCodeRegister(),
			want: []byte{0x44, 0xc2},
		},
	} {
		instruction, err := codec.BuildInstruction(c, "move.w", cpu68000parser.BinaryOperands(cpu68000.SizeWord, test.src, test.dst))
		assert.NoError(t, err)
		result, err := c.Assemble(t.Context(), []ast.Node{instruction})
		assert.NoError(t, err)
		assert.Equal(t, test.want, result.Binary)
	}
}

func TestCPU68000RejectsInvalidStatusInstructions(t *testing.T) {
	for _, line := range []string{
		"move.w ccr,d0", "move.b sr,d0", "move.l d0,sr", "move.w sr,a0",
		"move.w a0,ccr", "move.w sr,ccr", "move.w sr,#0", "move.w sr,4(pc)",
		"ori.w #1,ccr", "andi.b #1,sr", "addi.b #1,ccr", "subi.w #1,sr",
	} {
		t.Run(line, func(t *testing.T) {
			c := newCPU68000Codec(t)
			instruction, err := c.ParseInstruction(t.Context(), strings.NewReader(line))
			if err == nil {
				_, err = c.Assemble(t.Context(), []ast.Node{instruction})
			}
			assert.Error(t, err)
		})
	}
}
