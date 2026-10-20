package codec_test

import (
	"strings"
	"testing"

	chip8parser "github.com/retroenv/retroasm/pkg/arch/chip8/parser"
	"github.com/retroenv/retroasm/pkg/codec"
	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/chip8"
	"github.com/retroenv/retrogolib/assert"
)

func TestCHIP8Codec_DotSymbolLookupRoundTrip(t *testing.T) {
	t.Parallel()
	c := newCHIP8Codec(t)
	load, err := codec.BuildInstruction(c, chip8.LdName, chip8parser.Operands{
		chip8parser.SpecialOperand(chip8parser.OperandI),
		chip8parser.AddressOperand(ast.NewLabel(".Lretrogo_rodata_table")),
	})
	assert.NoError(t, err)
	add, err := codec.BuildInstruction(c, chip8.AddName, chip8parser.Operands{
		chip8parser.SpecialOperand(chip8parser.OperandI), chip8parser.RegisterOperand(0),
	})
	assert.NoError(t, err)
	read, err := codec.BuildInstruction(c, chip8.LdName, chip8parser.Operands{
		chip8parser.RegisterOperand(0), chip8parser.SpecialOperand(chip8parser.OperandIndirectI),
	})
	assert.NoError(t, err)
	jump, err := codec.BuildInstruction(c, chip8.JpName, chip8parser.Operands{
		chip8parser.AddressOperand(ast.NewLabel(".Ldone")),
	})
	assert.NoError(t, err)
	ret, err := codec.BuildInstruction(c, chip8.RetName, chip8parser.Operands{})
	assert.NoError(t, err)
	data := ast.NewData(ast.DataType, 1)
	data.Values = []*expression.Expression{
		expression.New(token.Token{
			Type:  token.Number,
			Value: "3",
		}),
		expression.New(token.Token{
			Type:  token.Number,
			Value: "9",
		}),
	}
	nodes := []ast.Node{load, add, read, jump, ast.NewLabel(".Lretrogo_rodata_table"), data, ast.NewLabel(".Ldone"), ret}
	typed, err := c.Assemble(t.Context(), nodes)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xa2, 0x08, 0xf0, 0x1e, 0xf0, 0x65, 0x12, 0x0a, 3, 9, 0, 0xee}, typed.Binary)
	formatted, err := c.FormatStream(ast.NewStreamFromNodes(nodes...))
	assert.NoError(t, err)
	parsed, err := c.ParseStream(t.Context(), "lookup.asm", strings.NewReader(formatted))
	assert.NoError(t, err)
	text, err := c.AssembleStream(t.Context(), parsed)
	assert.NoError(t, err)
	assert.Equal(t, typed.Binary, text.Binary)
	assert.Equal(t, typed.Symbols, text.Symbols)
}

func TestCHIP8Codec_DotAddressFormsRoundTrip(t *testing.T) {
	t.Parallel()
	for _, source := range []string{"jp .Ltarget", "jp v0,.Ltarget", "call .Ltarget", "ld i,.Ltarget"} {
		t.Run(source, func(t *testing.T) {
			t.Parallel()
			c := newCHIP8Codec(t)
			stream, err := c.ParseStream(t.Context(), "symbol.asm", strings.NewReader(source+"\n.Ltarget:\nret\n"))
			assert.NoError(t, err)
			before, err := c.AssembleStream(t.Context(), stream)
			assert.NoError(t, err)
			assert.Equal(t, uint64(0x202), before.Symbols[".Ltarget"])
			formatted, err := c.FormatStream(stream)
			assert.NoError(t, err)
			parsed, err := c.ParseStream(t.Context(), "formatted.asm", strings.NewReader(formatted))
			assert.NoError(t, err)
			after, err := c.AssembleStream(t.Context(), parsed)
			assert.NoError(t, err)
			assert.Equal(t, before.Binary, after.Binary)
		})
	}
}

func TestCHIP8Codec_DotOperandsRejectInvalidAddresses(t *testing.T) {
	t.Parallel()
	for _, source := range []string{"jp .", "ld i,.1", "jp .Lmissing", "ld i,0x1000"} {
		t.Run(source, func(t *testing.T) {
			t.Parallel()
			c := newCHIP8Codec(t)
			stream, err := c.ParseStream(t.Context(), "invalid.asm", strings.NewReader(source))
			if err == nil {
				_, err = c.AssembleStream(t.Context(), stream)
			}
			assert.Error(t, err)
		})
	}
}
