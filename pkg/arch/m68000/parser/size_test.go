package parser

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retrogolib/arch/cpu/m68000"
	"github.com/retroenv/retrogolib/assert"
)

var parseSizeSuffixTests = []struct {
	mnemonic string
	wantBase string
	wantSize m68000.OperandSize
}{
	{"MOVE.L", "MOVE", m68000.SizeLong},
	{"ADD.B", "ADD", m68000.SizeByte},
	{"CLR.W", "CLR", m68000.SizeWord},
	{"move.l", "move", m68000.SizeLong},
	// No suffix
	{"NOP", "NOP", 0},
	{"MOVE", "MOVE", 0},
	// Unknown suffix — returns original mnemonic unchanged
	{"MOVE.X", "MOVE.X", 0},
	// Dot but single char not B/W/L
	{"MOVE.Q", "MOVE.Q", 0},
}

func TestParseSizeSuffix(t *testing.T) {
	for _, tt := range parseSizeSuffixTests {
		t.Run(tt.mnemonic, func(t *testing.T) {
			base, size := ParseSizeSuffix(tt.mnemonic)
			assert.Equal(t, tt.wantBase, base)
			assert.Equal(t, tt.wantSize, size)
		})
	}
}

var parseSizeTokenTests = []struct {
	tok      token.Token
	wantSize m68000.OperandSize
}{
	{token.Token{Type: token.Identifier, Value: "B"}, m68000.SizeByte},
	{token.Token{Type: token.Identifier, Value: "W"}, m68000.SizeWord},
	{token.Token{Type: token.Identifier, Value: "L"}, m68000.SizeLong},
	{token.Token{Type: token.Identifier, Value: "b"}, m68000.SizeByte},
	{token.Token{Type: token.Identifier, Value: "w"}, m68000.SizeWord},
	{token.Token{Type: token.Identifier, Value: "l"}, m68000.SizeLong},
	// Non-size identifier
	{token.Token{Type: token.Identifier, Value: "X"}, 0},
	// Non-identifier token
	{token.Token{Type: token.Number, Value: "1"}, 0},
	{token.Token{Type: token.EOL}, 0},
}

func TestParseSizeToken(t *testing.T) {
	for _, tt := range parseSizeTokenTests {
		t.Run(tt.tok.Value, func(t *testing.T) {
			got := parseSizeToken(tt.tok)
			assert.Equal(t, tt.wantSize, got)
		})
	}
}
