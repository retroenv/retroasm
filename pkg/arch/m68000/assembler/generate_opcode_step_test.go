package assembler

import (
	"testing"

	m68000parser "github.com/retroenv/retroasm/pkg/arch/m68000/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/m68000"
	"github.com/retroenv/retrogolib/assert"
)

var opcodeEncodingTests = []struct {
	name     string
	address  uint64
	resolved m68000parser.ResolvedInstruction
	values   map[string]uint64
	want     []byte
}{
	// No-operand instructions
	{
		name:     "NOP",
		resolved: m68000parser.ResolvedInstruction{Instruction: m68000.Instructions[m68000.NOPName]},
		want:     []byte{0x4E, 0x71},
	},
	{
		name:     "RTS",
		resolved: m68000parser.ResolvedInstruction{Instruction: m68000.Instructions[m68000.RTSName]},
		want:     []byte{0x4E, 0x75},
	},
	{
		name:     "RTE",
		resolved: m68000parser.ResolvedInstruction{Instruction: m68000.Instructions[m68000.RTEName]},
		want:     []byte{0x4E, 0x73},
	},
	{
		name:     "RTR",
		resolved: m68000parser.ResolvedInstruction{Instruction: m68000.Instructions[m68000.RTRName]},
		want:     []byte{0x4E, 0x77},
	},
	{
		name:     "RESET",
		resolved: m68000parser.ResolvedInstruction{Instruction: m68000.Instructions[m68000.RESETName]},
		want:     []byte{0x4E, 0x70},
	},
	{
		name:     "TRAPV",
		resolved: m68000parser.ResolvedInstruction{Instruction: m68000.Instructions[m68000.TRAPVName]},
		want:     []byte{0x4E, 0x76},
	},
	{
		name:     "ILLEGAL",
		resolved: m68000parser.ResolvedInstruction{Instruction: m68000.Instructions[m68000.ILLEGALName]},
		want:     []byte{0x4A, 0xFC},
	},
	// MOVEQ #imm,Dn — 0111 Dn 0 data8
	{
		name: "MOVEQ #$42,D0",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.MOVEQName],
			Size:        m68000.SizeLong,
			SrcEA:       &m68000parser.EffectiveAddress{Mode: m68000.ImmediateMode, Value: ast.NewNumber(0x42)},
			DstEA:       &m68000parser.EffectiveAddress{Mode: m68000.DataRegDirectMode, Register: 0},
		},
		want: []byte{0x70, 0x42},
	},
	{
		name: "MOVEQ #0,D3",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.MOVEQName],
			Size:        m68000.SizeLong,
			SrcEA:       &m68000parser.EffectiveAddress{Mode: m68000.ImmediateMode, Value: ast.NewNumber(0)},
			DstEA:       &m68000parser.EffectiveAddress{Mode: m68000.DataRegDirectMode, Register: 3},
		},
		want: []byte{0x76, 0x00},
	},
	// MOVE.L D0,D1 — line 2 (long), src=D0(mode=0,reg=0), dst=D1(reg=1,mode=0)
	// 0010 001 000 000 000 = 0x2200
	{
		name: "MOVE.L D0,D1",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.MOVEName],
			Size:        m68000.SizeLong,
			SrcEA:       &m68000parser.EffectiveAddress{Mode: m68000.DataRegDirectMode, Register: 0},
			DstEA:       &m68000parser.EffectiveAddress{Mode: m68000.DataRegDirectMode, Register: 1},
		},
		want: []byte{0x22, 0x00},
	},
	// CLR.L D0 — 0100 0010 10 000 000 = 0x4280
	{
		name: "CLR.L D0",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.CLRName],
			Size:        m68000.SizeLong,
			DstEA:       &m68000parser.EffectiveAddress{Mode: m68000.DataRegDirectMode, Register: 0},
		},
		want: []byte{0x42, 0x80},
	},
	// SWAP D3 — 0x4840 | 3 = 0x4843
	{
		name: "SWAP D3",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.SWAPName],
			DstEA:       &m68000parser.EffectiveAddress{Mode: m68000.DataRegDirectMode, Register: 3},
		},
		want: []byte{0x48, 0x43},
	},
	// EXT.W D2 — 0x4880 | 2 = 0x4882
	{
		name: "EXT.W D2",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.EXTName],
			Size:        m68000.SizeWord,
			DstEA:       &m68000parser.EffectiveAddress{Mode: m68000.DataRegDirectMode, Register: 2},
		},
		want: []byte{0x48, 0x82},
	},
	// UNLK A6 — 0x4E58 | 6 = 0x4E5E
	{
		name: "UNLK A6",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.UNLKName],
			DstEA:       &m68000parser.EffectiveAddress{Mode: m68000.AddrRegDirectMode, Register: 6},
		},
		want: []byte{0x4E, 0x5E},
	},
	// TRAP #15 — 0x4E40 | 0x0F = 0x4E4F
	{
		name: "TRAP #15",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.TRAPName],
			SrcEA:       &m68000parser.EffectiveAddress{Mode: m68000.ImmediateMode, Value: ast.NewNumber(15)},
		},
		want: []byte{0x4E, 0x4F},
	},
	// ADDQ.W #1,D0 — 0101 001 0 01 000 000 = 0x5240
	{
		name: "ADDQ.W #1,D0",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.ADDQName],
			Size:        m68000.SizeWord,
			SrcEA:       &m68000parser.EffectiveAddress{Mode: m68000.ImmediateMode, Value: ast.NewNumber(1)},
			DstEA:       &m68000parser.EffectiveAddress{Mode: m68000.DataRegDirectMode, Register: 0},
		},
		want: []byte{0x52, 0x40},
	},
	// BEQ label (byte displacement) — condition 7, BEQ = Bcc with code 7
	// 0110 0111 displacement = 0x67 disp
	{
		name:    "BEQ label (byte displacement)",
		address: 0x1000,
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.BccName],
			Size:        m68000.SizeByte,
			Extra:       7, // EQ condition
			DstEA:       &m68000parser.EffectiveAddress{Mode: m68000.PCDisplacementMode, Value: ast.NewLabel("loop")},
		},
		values: map[string]uint64{"loop": 0x1010},
		// disp = 0x1010 - (0x1000 + 2) = 0x000E
		want: []byte{0x67, 0x0E},
	},
}

func TestGenerateInstructionOpcode(t *testing.T) {
	for _, tt := range opcodeEncodingTests {
		t.Run(tt.name, func(t *testing.T) {
			pc := tt.address
			if pc == 0 {
				pc = 0x1000
			}
			assigner := &mockAssigner{
				pc:     pc,
				values: tt.values,
			}
			ins := &mockInstruction{
				name:     tt.resolved.Instruction.Name,
				address:  pc,
				argument: tt.resolved,
			}

			err := GenerateInstructionOpcode(assigner, ins)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, ins.Opcodes())
			assert.Len(t, tt.want, ins.Size())
		})
	}
}

func TestGenerateInstructionOpcode_InvalidArgument(t *testing.T) {
	assigner := &mockAssigner{pc: 0x1000}
	ins := &mockInstruction{
		name:     "NOP",
		argument: "not-a-resolved-instruction",
	}

	err := GenerateInstructionOpcode(assigner, ins)
	assert.Error(t, err)
}
