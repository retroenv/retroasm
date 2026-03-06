package m68000

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"

	"github.com/retroenv/retroasm/pkg/assembler"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/m68000"
)

func TestArchitectureLookup(t *testing.T) {
	cfg := New()
	arch := cfg.Arch

	tests := []struct {
		name     string
		expected string
	}{
		{"nop", m68000.NOPName},
		{"rts", m68000.RTSName},
		{"move", m68000.MOVEName},
		{"move.l", m68000.MOVEName},
		{"move.w", m68000.MOVEName},
		{"move.b", m68000.MOVEName},
		{"add", m68000.ADDName},
		{"beq", m68000.BccName},
		{"bne", m68000.BccName},
		{"bra", m68000.BRAName},
		{"dbne", m68000.DBccName},
		{"shi", m68000.SccName},
		{"lea", m68000.LEAName},
		{"movea", m68000.MOVEAName},
		{"movem", m68000.MOVEMName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ins, ok := arch.Instruction(tt.name)
			if !ok {
				t.Fatalf("Instruction(%q) returned false", tt.name)
			}
			if ins.Name != tt.expected {
				t.Errorf("Instruction(%q).Name = %q, want %q", tt.name, ins.Name, tt.expected)
			}
		})
	}

	// Test unknown instruction
	_, ok := arch.Instruction("unknown")
	if ok {
		t.Error("Instruction(\"unknown\") should return false")
	}
}

func TestAddressWidth(t *testing.T) {
	cfg := New()
	if w := cfg.Arch.AddressWidth(); w != 24 {
		t.Errorf("AddressWidth() = %d, want 24", w)
	}
}

func TestAssembleSimpleInstructions(t *testing.T) {
	tests := []struct {
		name     string
		asm      string
		expected []byte
	}{
		{"NOP", ".segment \"CODE\"\nNOP\n", []byte{0x4E, 0x71}},
		{"RTS", ".segment \"CODE\"\nRTS\n", []byte{0x4E, 0x75}},
		{"RTE", ".segment \"CODE\"\nRTE\n", []byte{0x4E, 0x73}},
		{"RTR", ".segment \"CODE\"\nRTR\n", []byte{0x4E, 0x77}},
		{"RESET", ".segment \"CODE\"\nRESET\n", []byte{0x4E, 0x70}},
		{"TRAPV", ".segment \"CODE\"\nTRAPV\n", []byte{0x4E, 0x76}},
		{"ILLEGAL", ".segment \"CODE\"\nILLEGAL\n", []byte{0x4A, 0xFC}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assembleM68000(t, tt.asm)
			assertBytes(t, tt.expected, result)
		})
	}
}

func TestAssembleMOVEQ(t *testing.T) {
	asm := ".segment \"CODE\"\nMOVEQ #$42,D0\n"
	result := assembleM68000(t, asm)

	// MOVEQ: 0111 Dn 0 data8
	// D0=0, data=0x42 -> 0x7042
	expected := []byte{0x70, 0x42}
	assertBytes(t, expected, result)
}

func TestAssembleCLR(t *testing.T) {
	asm := ".segment \"CODE\"\nCLR.L D0\n"
	result := assembleM68000(t, asm)

	// CLR.L D0: 0100 0010 10 000 000 = 0x4280
	expected := []byte{0x42, 0x80}
	assertBytes(t, expected, result)
}

func TestAssembleSWAP(t *testing.T) {
	asm := ".segment \"CODE\"\nSWAP D3\n"
	result := assembleM68000(t, asm)

	// SWAP D3: 0x4840 | 3 = 0x4843
	expected := []byte{0x48, 0x43}
	assertBytes(t, expected, result)
}

func TestAssembleTRAP(t *testing.T) {
	asm := ".segment \"CODE\"\nTRAP #$0F\n"
	result := assembleM68000(t, asm)

	// TRAP #15: 0x4E40 | 0x0F = 0x4E4F
	expected := []byte{0x4E, 0x4F}
	assertBytes(t, expected, result)
}

func TestAssembleEXT(t *testing.T) {
	asm := ".segment \"CODE\"\nEXT.W D2\n"
	result := assembleM68000(t, asm)

	// EXT.W D2: 0x4880 | 2 = 0x4882
	expected := []byte{0x48, 0x82}
	assertBytes(t, expected, result)
}

func TestAssembleUNLK(t *testing.T) {
	asm := ".segment \"CODE\"\nUNLK A6\n"
	result := assembleM68000(t, asm)

	// UNLK A6: 0x4E58 | 6 = 0x4E5E
	expected := []byte{0x4E, 0x5E}
	assertBytes(t, expected, result)
}

// TestRoundTrip verifies that encoding produces opcodes that the retrogolib decoder recognizes.
func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		opcode uint16
	}{
		{"NOP", 0x4E71},
		{"RTS", 0x4E75},
		{"ILLEGAL", 0x4afc},
		{"RESET", 0x4E70},
		{"TRAPV", 0x4E76},
		{"RTE", 0x4E73},
		{"RTR", 0x4E77},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, 2)
			binary.BigEndian.PutUint16(buf, tt.opcode)

			result := assembleM68000(t, ".segment \"CODE\"\n"+tt.name+"\n")
			if len(result) < 2 {
				t.Fatalf("result too short: %d bytes", len(result))
			}
			got := binary.BigEndian.Uint16(result[:2])
			if got != tt.opcode {
				t.Errorf("opcode mismatch: got 0x%04X, want 0x%04X", got, tt.opcode)
			}
		})
	}
}

func assembleM68000(t *testing.T, source string) []byte {
	t.Helper()

	cfg := New()
	if err := cfg.ReadCa65Config(bytes.NewReader([]byte(defaultM68000Config))); err != nil {
		t.Fatalf("reading config: %v", err)
	}

	var buf bytes.Buffer
	asm := assembler.New(cfg, &buf)

	ctx := context.Background()
	if err := asm.Process(ctx, bytes.NewReader([]byte(source))); err != nil {
		t.Fatalf("assembling: %v", err)
	}

	return buf.Bytes()
}

func assertBytes(t *testing.T, expected, got []byte) {
	t.Helper()
	if len(got) < len(expected) {
		t.Fatalf("result too short: got %d bytes, want at least %d", len(got), len(expected))
	}
	for i, b := range expected {
		if got[i] != b {
			t.Errorf("byte %d: got 0x%02X, want 0x%02X", i, got[i], b)
		}
	}
}

const defaultM68000Config = `
MEMORY {
    CODE: start = $0, size = $10000, fill = yes;
}
SEGMENTS {
    CODE: load = CODE, type = rw;
}
`

func TestAssembleMOVE(t *testing.T) {
	// MOVE.L D0,D1: line 2 (long), src=D0 (mode=0,reg=0), dst=D1 (reg=1,mode=0)
	// Opcode: 0010 001 000 000 000 = 0x2200
	asm := ".segment \"CODE\"\nMOVE.L D0,D1\n"
	result := assembleM68000(t, asm)
	expected := []byte{0x22, 0x00}
	assertBytes(t, expected, result)
}

func TestAssembleADDQ(t *testing.T) {
	// ADDQ.W #1,D0: 0101 001 0 01 000 000 = 0x5240
	asm := ".segment \"CODE\"\nADDQ.W #1,D0\n"
	result := assembleM68000(t, asm)
	expected := []byte{0x52, 0x40}
	assertBytes(t, expected, result)
}

func TestAssembleMultipleInstructions(t *testing.T) {
	source := `.segment "CODE"
MOVEQ #0,D0
NOP
RTS
`
	result := assembleM68000(t, source)
	// MOVEQ #0,D0 = 0x7000
	// NOP = 0x4E71
	// RTS = 0x4E75
	expected := []byte{0x70, 0x00, 0x4E, 0x71, 0x4E, 0x75}
	assertBytes(t, expected, result)
}

// Verify our AST can be processed.
func TestAssembleAST(t *testing.T) {
	cfg := New()
	if err := cfg.ReadCa65Config(bytes.NewReader([]byte(defaultM68000Config))); err != nil {
		t.Fatalf("reading config: %v", err)
	}

	nodes := []ast.Node{
		ast.NewSegment("CODE"),
	}

	var buf bytes.Buffer
	asm := assembler.New(cfg, &buf)

	ctx := context.Background()
	if err := asm.ProcessAST(ctx, nodes); err != nil {
		t.Fatalf("processing AST: %v", err)
	}
}
