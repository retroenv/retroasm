package codec_test

import (
	"strings"
	"testing"

	asmchip8 "github.com/retroenv/retroasm/pkg/arch/chip8"
	chip8parser "github.com/retroenv/retroasm/pkg/arch/chip8/parser"
	"github.com/retroenv/retroasm/pkg/codec"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/arch/cpu/chip8"
	"github.com/retroenv/retrogolib/assert"
)

//nolint:funlen // Addressing-family coverage is intentionally one auditable table.
func TestCHIP8Codec_AddressingFamiliesRoundTrip(t *testing.T) {
	t.Parallel()

	c := newCHIP8Codec(t)
	tests := []struct {
		name     string
		mnemonic string
		operands chip8parser.Operands
		wantText string
		wantCode []byte
	}{
		{name: "implied", mnemonic: chip8.ClsName, wantText: "cls", wantCode: []byte{0x00, 0xe0}},
		{
			name: "absolute", mnemonic: chip8.CallName,
			operands: chip8parser.Operands{chip8parser.AddressOperand(ast.NewNumber(0x400))},
			wantText: "call 0x400", wantCode: []byte{0x24, 0x00},
		},
		{
			name: "v0 absolute", mnemonic: chip8.JpName,
			operands: chip8parser.Operands{
				chip8parser.RegisterOperand(0),
				chip8parser.AddressOperand(ast.NewNumber(0x300)),
			},
			wantText: "jp v0,0x300", wantCode: []byte{0xb3, 0x00},
		},
		{
			name: "single register", mnemonic: chip8.SkpName,
			operands: chip8parser.Operands{chip8parser.RegisterOperand(1)},
			wantText: "skp v1", wantCode: []byte{0xe1, 0x9e},
		},
		{
			name: "register byte", mnemonic: chip8.LdName,
			operands: chip8parser.Operands{
				chip8parser.RegisterOperand(3),
				chip8parser.ByteOperand(ast.NewNumber(0x7f)),
			},
			wantText: "ld v3,0x7F", wantCode: []byte{0x63, 0x7f},
		},
		{
			name: "register pair", mnemonic: chip8.XorName,
			operands: chip8parser.Operands{chip8parser.RegisterOperand(5), chip8parser.RegisterOperand(6)},
			wantText: "xor v5,v6", wantCode: []byte{0x85, 0x63},
		},
		{
			name: "register pair nibble", mnemonic: chip8.DrwName,
			operands: chip8parser.Operands{
				chip8parser.RegisterOperand(1),
				chip8parser.RegisterOperand(2),
				chip8parser.NibbleOperand(ast.NewNumber(5)),
			},
			wantText: "drw v1,v2,0x5", wantCode: []byte{0xd1, 0x25},
		},
		{
			name: "register delay timer", mnemonic: chip8.LdName,
			operands: chip8parser.Operands{
				chip8parser.RegisterOperand(2), chip8parser.SpecialOperand(chip8parser.OperandDT),
			},
			wantText: "ld v2,dt", wantCode: []byte{0xf2, 0x07},
		},
		{
			name: "register key", mnemonic: chip8.LdName,
			operands: chip8parser.Operands{
				chip8parser.RegisterOperand(0), chip8parser.SpecialOperand(chip8parser.OperandK),
			},
			wantText: "ld v0,k", wantCode: []byte{0xf0, 0x0a},
		},
		{
			name: "delay timer register", mnemonic: chip8.LdName,
			operands: chip8parser.Operands{
				chip8parser.SpecialOperand(chip8parser.OperandDT), chip8parser.RegisterOperand(5),
			},
			wantText: "ld dt,v5", wantCode: []byte{0xf5, 0x15},
		},
		{
			name: "sound timer register", mnemonic: chip8.LdName,
			operands: chip8parser.Operands{
				chip8parser.SpecialOperand(chip8parser.OperandST), chip8parser.RegisterOperand(3),
			},
			wantText: "ld st,v3", wantCode: []byte{0xf3, 0x18},
		},
		{
			name: "font register", mnemonic: chip8.LdName,
			operands: chip8parser.Operands{
				chip8parser.SpecialOperand(chip8parser.OperandF), chip8parser.RegisterOperand(7),
			},
			wantText: "ld f,v7", wantCode: []byte{0xf7, 0x29},
		},
		{
			name: "bcd register", mnemonic: chip8.LdName,
			operands: chip8parser.Operands{
				chip8parser.SpecialOperand(chip8parser.OperandB), chip8parser.RegisterOperand(0xa),
			},
			wantText: "ld b,va", wantCode: []byte{0xfa, 0x33},
		},
		{
			name: "i absolute", mnemonic: chip8.LdName,
			operands: chip8parser.Operands{
				chip8parser.SpecialOperand(chip8parser.OperandI),
				chip8parser.AddressOperand(ast.NewNumber(0x300)),
			},
			wantText: "ld i,0x300", wantCode: []byte{0xa3, 0x00},
		},
		{
			name: "i register", mnemonic: chip8.AddName,
			operands: chip8parser.Operands{
				chip8parser.SpecialOperand(chip8parser.OperandI), chip8parser.RegisterOperand(1),
			},
			wantText: "add i,v1", wantCode: []byte{0xf1, 0x1e},
		},
		{
			name: "indirect i register", mnemonic: chip8.LdName,
			operands: chip8parser.Operands{
				chip8parser.SpecialOperand(chip8parser.OperandIndirectI), chip8parser.RegisterOperand(1),
			},
			wantText: "ld [i],v1", wantCode: []byte{0xf1, 0x55},
		},
		{
			name: "register indirect i", mnemonic: chip8.LdName,
			operands: chip8parser.Operands{
				chip8parser.RegisterOperand(1), chip8parser.SpecialOperand(chip8parser.OperandIndirectI),
			},
			wantText: "ld v1,[i]", wantCode: []byte{0xf1, 0x65},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			built, err := codec.BuildInstruction(c, test.mnemonic, test.operands)
			assert.NoError(t, err)
			assert.True(t, built.OpcodeID.ValidFor(arch.CHIP8))
			assert.NoError(t, c.ValidateInstruction(built))

			formatted, err := c.FormatInstruction(built)
			assert.NoError(t, err)
			assert.Equal(t, test.wantText, formatted)

			parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
			assert.NoError(t, err)
			assert.NoError(t, c.ValidateInstruction(parsed))

			builtAssembly, err := c.Assemble(t.Context(), []ast.Node{built})
			assert.NoError(t, err)
			parsedAssembly, err := c.Assemble(t.Context(), []ast.Node{parsed})
			assert.NoError(t, err)
			assert.Equal(t, test.wantCode, builtAssembly.Binary)
			assert.Equal(t, builtAssembly.Binary, parsedAssembly.Binary)
		})
	}
}

func TestCHIP8Codec_SymbolicExpressionRoundTrip(t *testing.T) {
	t.Parallel()

	c := newCHIP8Codec(t)
	instruction, err := codec.BuildInstruction(
		c,
		chip8.LdName,
		chip8parser.Operands{
			chip8parser.RegisterOperand(0),
			chip8parser.ByteOperand(ast.NewExpression(
				astToken("slot"), astOperatorToken("%"), astNumberToken("256"),
			)),
		},
	)
	assert.NoError(t, err)
	formatted, err := c.FormatInstruction(instruction)
	assert.NoError(t, err)
	assert.Equal(t, "ld v0,slot % 0x100", formatted)
	parsed, err := c.ParseInstruction(t.Context(), strings.NewReader(formatted))
	assert.NoError(t, err)
	assert.NoError(t, c.ValidateInstruction(parsed))
}

func TestCHIP8Codec_RejectsInvalidTypedOperands(t *testing.T) {
	t.Parallel()

	c := newCHIP8Codec(t)
	tests := []chip8parser.Operands{
		{chip8parser.RegisterOperand(16), chip8parser.ByteOperand(ast.NewNumber(1))},
		{chip8parser.RegisterOperand(1), chip8parser.ByteOperand(ast.NewNumber(0x100))},
		{chip8parser.SpecialOperand(chip8parser.OperandI), chip8parser.AddressOperand(ast.NewNumber(0x1000))},
	}
	for _, operands := range tests {
		_, err := codec.BuildInstruction(c, chip8.LdName, operands)
		assert.Error(t, err)
	}
}

func TestCHIP8TypedInstructionFormattingOptions(t *testing.T) {
	t.Parallel()

	c := newCHIP8Codec(t)
	instruction, err := codec.BuildInstruction(
		c,
		chip8.LdName,
		chip8parser.Operands{chip8parser.RegisterOperand(0xa), chip8parser.ByteOperand(ast.NewNumber(1))},
	)
	assert.NoError(t, err)
	formatted, err := chip8parser.FormatInstructionWithOptions(
		instruction,
		chip8parser.FormatOptions{Indent: "  ", Uppercase: true},
	)
	assert.NoError(t, err)
	assert.Equal(t, "  LD VA,0x01", formatted)
}

func newCHIP8Codec(t *testing.T) *codec.Codec[*chip8.Instruction] {
	t.Helper()
	c, err := codec.New(asmchip8.New())
	assert.NoError(t, err)
	return c
}

func astToken(value string) token.Token {
	return token.Token{Type: token.Identifier, Value: value}
}

func astOperatorToken(value string) token.Token {
	return token.Token{Type: token.Percent, Value: value}
}

func astNumberToken(value string) token.Token {
	return token.Token{Type: token.Number, Value: value}
}
