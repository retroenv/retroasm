package m65816

import (
	"bytes"
	"context"
	"testing"

	"github.com/retroenv/retroasm/pkg/assembler"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/m65816"
	"github.com/retroenv/retrogolib/assert"
)

func TestArchitectureLookup(t *testing.T) {
	cfg := New()
	a := cfg.Arch

	tests := []struct {
		name     string
		expected string
	}{
		{"clc", m65816.ClcName},
		{"lda", m65816.LdaName},
		{"nop", m65816.NopName},
		{"rtl", m65816.RtlName},
		{"rts", m65816.RtsName},
		{"sta", m65816.StaName},
		{"xba", m65816.XbaName},
		{"xce", m65816.XceName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ins, ok := a.Instruction(tt.name)
			assert.True(t, ok)
			assert.Equal(t, tt.expected, ins.Name)
		})
	}

	_, ok := a.Instruction("unknown")
	assert.False(t, ok)
}

func TestAddressWidth(t *testing.T) {
	cfg := New()
	assert.Equal(t, 24, cfg.Arch.AddressWidth())
}

func TestAssembleImpliedInstructions(t *testing.T) {
	tests := []struct {
		name     string
		asm      string
		expected []byte
	}{
		{"CLC", ".segment \"CODE\"\nCLC\n", []byte{0x18}},
		{"NOP", ".segment \"CODE\"\nNOP\n", []byte{0xEA}},
		{"RTL", ".segment \"CODE\"\nRTL\n", []byte{0x6B}},
		{"RTS", ".segment \"CODE\"\nRTS\n", []byte{0x60}},
		{"SEC", ".segment \"CODE\"\nSEC\n", []byte{0x38}},
		{"SEI", ".segment \"CODE\"\nSEI\n", []byte{0x78}},
		{"STP", ".segment \"CODE\"\nSTP\n", []byte{0xDB}},
		{"WAI", ".segment \"CODE\"\nWAI\n", []byte{0xCB}},
		{"XBA", ".segment \"CODE\"\nXBA\n", []byte{0xEB}},
		{"XCE", ".segment \"CODE\"\nXCE\n", []byte{0xFB}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assembleM65816(t, tt.asm)
			assertBytes(t, tt.expected, result)
		})
	}
}

func TestAssembleImmediateAddressing(t *testing.T) {
	tests := []struct {
		name     string
		asm      string
		expected []byte
	}{
		{"LDA #$42", ".segment \"CODE\"\nLDA #$42\n", []byte{0xA9, 0x42}},
		{"LDX #$00", ".segment \"CODE\"\nLDX #$00\n", []byte{0xA2, 0x00}},
		{"LDY #$FF", ".segment \"CODE\"\nLDY #$FF\n", []byte{0xA0, 0xFF}},
		{"REP #$30", ".segment \"CODE\"\nREP #$30\n", []byte{0xC2, 0x30}},
		{"SEP #$20", ".segment \"CODE\"\nSEP #$20\n", []byte{0xE2, 0x20}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assembleM65816(t, tt.asm)
			assertBytes(t, tt.expected, result)
		})
	}
}

func TestAssembleDirectPageAddressing(t *testing.T) {
	tests := []struct {
		name     string
		asm      string
		expected []byte
	}{
		{"LDA $10", ".segment \"CODE\"\nLDA $10\n", []byte{0xA5, 0x10}},
		{"STA $80", ".segment \"CODE\"\nSTA $80\n", []byte{0x85, 0x80}},
		{"LDA $20,X", ".segment \"CODE\"\nLDA $20,X\n", []byte{0xB5, 0x20}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assembleM65816(t, tt.asm)
			assertBytes(t, tt.expected, result)
		})
	}
}

func TestAssembleAbsoluteAddressing(t *testing.T) {
	tests := []struct {
		name     string
		asm      string
		expected []byte
	}{
		{"LDA $1234", ".segment \"CODE\"\nLDA $1234\n", []byte{0xAD, 0x34, 0x12}},
		{"STA $2000", ".segment \"CODE\"\nSTA $2000\n", []byte{0x8D, 0x00, 0x20}},
		{"JMP $8000", ".segment \"CODE\"\nJMP $8000\n", []byte{0x4C, 0x00, 0x80}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assembleM65816(t, tt.asm)
			assertBytes(t, tt.expected, result)
		})
	}
}

func TestAssembleAbsoluteLongAddressing(t *testing.T) {
	tests := []struct {
		name     string
		asm      string
		expected []byte
	}{
		{"JML $12ABCD", ".segment \"CODE\"\nJML $12ABCD\n", []byte{0x5C, 0xCD, 0xAB, 0x12}},
		{"JSL $018000", ".segment \"CODE\"\nJSL $018000\n", []byte{0x22, 0x00, 0x80, 0x01}},
		{"LDA f:$012345", ".segment \"CODE\"\nLDA f:$012345\n", []byte{0xAF, 0x45, 0x23, 0x01}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assembleM65816(t, tt.asm)
			assertBytes(t, tt.expected, result)
		})
	}
}

func TestAssembleAccumulatorAddressing(t *testing.T) {
	tests := []struct {
		name     string
		asm      string
		expected []byte
	}{
		{"ASL A", ".segment \"CODE\"\nASL A\n", []byte{0x0A}},
		{"LSR A", ".segment \"CODE\"\nLSR A\n", []byte{0x4A}},
		{"ROL A", ".segment \"CODE\"\nROL A\n", []byte{0x2A}},
		{"ROR A", ".segment \"CODE\"\nROR A\n", []byte{0x6A}},
		{"INC A", ".segment \"CODE\"\nINC A\n", []byte{0x1A}},
		{"DEC A", ".segment \"CODE\"\nDEC A\n", []byte{0x3A}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assembleM65816(t, tt.asm)
			assertBytes(t, tt.expected, result)
		})
	}
}

func TestAssembleIndirectLongAddressing(t *testing.T) {
	tests := []struct {
		name     string
		asm      string
		expected []byte
	}{
		{"LDA [$10]", ".segment \"CODE\"\nLDA [$10]\n", []byte{0xA7, 0x10}},
		{"LDA [$20],Y", ".segment \"CODE\"\nLDA [$20],Y\n", []byte{0xB7, 0x20}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assembleM65816(t, tt.asm)
			assertBytes(t, tt.expected, result)
		})
	}
}

func TestAssembleStackRelativeAddressing(t *testing.T) {
	tests := []struct {
		name     string
		asm      string
		expected []byte
	}{
		{"LDA $05,S", ".segment \"CODE\"\nLDA $05,S\n", []byte{0xA3, 0x05}},
		{"STA $03,S", ".segment \"CODE\"\nSTA $03,S\n", []byte{0x83, 0x03}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assembleM65816(t, tt.asm)
			assertBytes(t, tt.expected, result)
		})
	}
}

func TestAssembleStackRelativeIndirectIndexedY(t *testing.T) {
	tests := []struct {
		name     string
		asm      string
		expected []byte
	}{
		{"LDA ($05,S),Y", ".segment \"CODE\"\nLDA ($05,S),Y\n", []byte{0xB3, 0x05}},
		{"STA ($03,S),Y", ".segment \"CODE\"\nSTA ($03,S),Y\n", []byte{0x93, 0x03}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assembleM65816(t, tt.asm)
			assertBytes(t, tt.expected, result)
		})
	}
}

func TestAssembleBlockMove(t *testing.T) {
	tests := []struct {
		name     string
		asm      string
		expected []byte
	}{
		{"MVN $01,$02", ".segment \"CODE\"\nMVN $01,$02\n", []byte{0x54, 0x02, 0x01}},
		{"MVP $7E,$7F", ".segment \"CODE\"\nMVP $7E,$7F\n", []byte{0x44, 0x7F, 0x7E}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assembleM65816(t, tt.asm)
			assertBytes(t, tt.expected, result)
		})
	}
}

func TestAssembleBranchInstructions(t *testing.T) {
	// BRA to self (offset -2): opcode + branch byte
	asm := ".segment \"CODE\"\nloop:\nBRA loop\n"
	result := assembleM65816(t, asm)
	assertBytes(t, []byte{0x80, 0xFE}, result)
}

func TestAssembleBRL(t *testing.T) {
	// BRL to self: opcode + 16-bit offset (-3 = 0xFFFD)
	asm := ".segment \"CODE\"\nloop:\nBRL loop\n"
	result := assembleM65816(t, asm)
	assertBytes(t, []byte{0x82, 0xFD, 0xFF}, result)
}

func TestAssembleMultipleInstructions(t *testing.T) {
	source := `.segment "CODE"
CLC
XCE
SEI
NOP
RTS
`
	result := assembleM65816(t, source)
	expected := []byte{0x18, 0xFB, 0x78, 0xEA, 0x60}
	assertBytes(t, expected, result)
}

func TestAssembleAST(t *testing.T) {
	cfg := New()
	assert.NoError(t, cfg.ReadCa65Config(bytes.NewReader([]byte(defaultM65816Config))))

	nodes := []ast.Node{
		ast.NewSegment("CODE"),
	}

	var buf bytes.Buffer
	asm := assembler.New(cfg, &buf)

	ctx := context.Background()
	assert.NoError(t, asm.ProcessAST(ctx, nodes))
}

func TestAssembleIndirectAddressing(t *testing.T) {
	tests := []struct {
		name     string
		asm      string
		expected []byte
	}{
		{"LDA ($10)", ".segment \"CODE\"\nLDA ($10)\n", []byte{0xB2, 0x10}},
		{"LDA ($20,X)", ".segment \"CODE\"\nLDA ($20,X)\n", []byte{0xA1, 0x20}},
		{"LDA ($30),Y", ".segment \"CODE\"\nLDA ($30),Y\n", []byte{0xB1, 0x30}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assembleM65816(t, tt.asm)
			assertBytes(t, tt.expected, result)
		})
	}
}

func assembleM65816(t *testing.T, source string) []byte {
	t.Helper()

	cfg := New()
	assert.NoError(t, cfg.ReadCa65Config(bytes.NewReader([]byte(defaultM65816Config))))

	var buf bytes.Buffer
	asm := assembler.New(cfg, &buf)

	ctx := context.Background()
	assert.NoError(t, asm.Process(ctx, bytes.NewReader([]byte(source))))

	return buf.Bytes()
}

func assertBytes(t *testing.T, expected, got []byte) {
	t.Helper()
	assert.GreaterOrEqual(t, len(got), len(expected))
	for i, b := range expected {
		assert.Equal(t, b, got[i])
	}
}

const defaultM65816Config = `
MEMORY {
    CODE: start = $0, size = $1000000, fill = yes;
}
SEGMENTS {
    CODE: load = CODE, type = rw;
}
`
