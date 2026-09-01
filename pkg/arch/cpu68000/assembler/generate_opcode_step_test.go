package assembler

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/arch/cpu68000/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu68000"
	"github.com/retroenv/retrogolib/assert"
)

var opcodeEncodingTests = []struct {
	name     string
	address  uint64
	resolved parser.ResolvedInstruction
	values   map[string]uint64
	want     []byte
}{
	// No-operand instructions
	{
		name:     "NOP",
		resolved: parser.ResolvedInstruction{Instruction: cpu68000.Instructions[cpu68000.NOPName]},
		want:     []byte{0x4E, 0x71},
	},
	{
		name:     "RTS",
		resolved: parser.ResolvedInstruction{Instruction: cpu68000.Instructions[cpu68000.RTSName]},
		want:     []byte{0x4E, 0x75},
	},
	{
		name:     "RTE",
		resolved: parser.ResolvedInstruction{Instruction: cpu68000.Instructions[cpu68000.RTEName]},
		want:     []byte{0x4E, 0x73},
	},
	{
		name:     "RTR",
		resolved: parser.ResolvedInstruction{Instruction: cpu68000.Instructions[cpu68000.RTRName]},
		want:     []byte{0x4E, 0x77},
	},
	{
		name:     "RESET",
		resolved: parser.ResolvedInstruction{Instruction: cpu68000.Instructions[cpu68000.RESETName]},
		want:     []byte{0x4E, 0x70},
	},
	{
		name:     "TRAPV",
		resolved: parser.ResolvedInstruction{Instruction: cpu68000.Instructions[cpu68000.TRAPVName]},
		want:     []byte{0x4E, 0x76},
	},
	{
		name:     "ILLEGAL",
		resolved: parser.ResolvedInstruction{Instruction: cpu68000.Instructions[cpu68000.ILLEGALName]},
		want:     []byte{0x4A, 0xFC},
	},
	// MOVEQ #imm,Dn — 0111 Dn 0 data8
	{
		name: "MOVEQ #$42,D0",
		resolved: parser.ResolvedInstruction{
			Instruction: cpu68000.Instructions[cpu68000.MOVEQName],
			Size:        cpu68000.SizeLong,
			SrcEA:       &parser.EffectiveAddress{Mode: cpu68000.ImmediateMode, Value: ast.NewNumber(0x42)},
			DstEA:       &parser.EffectiveAddress{Mode: cpu68000.DataRegDirectMode, Register: 0},
		},
		want: []byte{0x70, 0x42},
	},
	{
		name: "MOVEQ #0,D3",
		resolved: parser.ResolvedInstruction{
			Instruction: cpu68000.Instructions[cpu68000.MOVEQName],
			Size:        cpu68000.SizeLong,
			SrcEA:       &parser.EffectiveAddress{Mode: cpu68000.ImmediateMode, Value: ast.NewNumber(0)},
			DstEA:       &parser.EffectiveAddress{Mode: cpu68000.DataRegDirectMode, Register: 3},
		},
		want: []byte{0x76, 0x00},
	},
	// MOVE.L D0,D1 — line 2 (long), src=D0(mode=0,reg=0), dst=D1(reg=1,mode=0)
	// 0010 001 000 000 000 = 0x2200
	{
		name: "MOVE.L D0,D1",
		resolved: parser.ResolvedInstruction{
			Instruction: cpu68000.Instructions[cpu68000.MOVEName],
			Size:        cpu68000.SizeLong,
			SrcEA:       &parser.EffectiveAddress{Mode: cpu68000.DataRegDirectMode, Register: 0},
			DstEA:       &parser.EffectiveAddress{Mode: cpu68000.DataRegDirectMode, Register: 1},
		},
		want: []byte{0x22, 0x00},
	},
	// CLR.L D0 — 0100 0010 10 000 000 = 0x4280
	{
		name: "CLR.L D0",
		resolved: parser.ResolvedInstruction{
			Instruction: cpu68000.Instructions[cpu68000.CLRName],
			Size:        cpu68000.SizeLong,
			DstEA:       &parser.EffectiveAddress{Mode: cpu68000.DataRegDirectMode, Register: 0},
		},
		want: []byte{0x42, 0x80},
	},
	// SWAP D3 — 0x4840 | 3 = 0x4843
	{
		name: "SWAP D3",
		resolved: parser.ResolvedInstruction{
			Instruction: cpu68000.Instructions[cpu68000.SWAPName],
			DstEA:       &parser.EffectiveAddress{Mode: cpu68000.DataRegDirectMode, Register: 3},
		},
		want: []byte{0x48, 0x43},
	},
	// EXT.W D2 — 0x4880 | 2 = 0x4882
	{
		name: "EXT.W D2",
		resolved: parser.ResolvedInstruction{
			Instruction: cpu68000.Instructions[cpu68000.EXTName],
			Size:        cpu68000.SizeWord,
			DstEA:       &parser.EffectiveAddress{Mode: cpu68000.DataRegDirectMode, Register: 2},
		},
		want: []byte{0x48, 0x82},
	},
	// UNLK A6 — 0x4E58 | 6 = 0x4E5E
	{
		name: "UNLK A6",
		resolved: parser.ResolvedInstruction{
			Instruction: cpu68000.Instructions[cpu68000.UNLKName],
			DstEA:       &parser.EffectiveAddress{Mode: cpu68000.AddrRegDirectMode, Register: 6},
		},
		want: []byte{0x4E, 0x5E},
	},
	// TRAP #15 — 0x4E40 | 0x0F = 0x4E4F
	{
		name: "TRAP #15",
		resolved: parser.ResolvedInstruction{
			Instruction: cpu68000.Instructions[cpu68000.TRAPName],
			SrcEA:       &parser.EffectiveAddress{Mode: cpu68000.ImmediateMode, Value: ast.NewNumber(15)},
		},
		want: []byte{0x4E, 0x4F},
	},
	// ADDQ.W #1,D0 — 0101 001 0 01 000 000 = 0x5240
	{
		name: "ADDQ.W #1,D0",
		resolved: parser.ResolvedInstruction{
			Instruction: cpu68000.Instructions[cpu68000.ADDQName],
			Size:        cpu68000.SizeWord,
			SrcEA:       &parser.EffectiveAddress{Mode: cpu68000.ImmediateMode, Value: ast.NewNumber(1)},
			DstEA:       &parser.EffectiveAddress{Mode: cpu68000.DataRegDirectMode, Register: 0},
		},
		want: []byte{0x52, 0x40},
	},
	// BEQ label (byte displacement) — condition 7, BEQ = Bcc with code 7
	// 0110 0111 displacement = 0x67 disp
	{
		name:    "BEQ label (byte displacement)",
		address: 0x1000,
		resolved: parser.ResolvedInstruction{
			Instruction: cpu68000.Instructions[cpu68000.BccName],
			Size:        cpu68000.SizeByte,
			Extra:       7, // EQ condition
			DstEA:       &parser.EffectiveAddress{Mode: cpu68000.PCDisplacementMode, Value: ast.NewLabel("loop")},
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
