package assembler

import (
	"testing"

	z80parser "github.com/retroenv/retroasm/pkg/arch/z80/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
	"github.com/retroenv/retrogolib/assert"
)

func TestGenerateInstructionOpcode_CoreEncodings(t *testing.T) { //nolint:funlen
	tests := []struct {
		name     string
		address  uint64
		resolved z80parser.ResolvedInstruction
		values   map[string]uint64
		want     []byte
	}{
		{
			name:    "nop",
			address: 0x8000,
			resolved: z80parser.ResolvedInstruction{
				Addressing:  cpuz80.ImpliedAddressing,
				Instruction: cpuz80.Nop,
			},
			want: []byte{0x00},
		},
		{
			name:    "ld bc,nn",
			address: 0x8000,
			resolved: z80parser.ResolvedInstruction{
				Addressing:     cpuz80.ImmediateAddressing,
				Instruction:    cpuz80.LdReg16,
				RegisterParams: []cpuz80.RegisterParam{cpuz80.RegBC},
				OperandValues:  []ast.Node{ast.NewNumber(0x1234)},
			},
			want: []byte{0x01, 0x34, 0x12},
		},
		{
			name:    "ld a,n",
			address: 0x8000,
			resolved: z80parser.ResolvedInstruction{
				Addressing:     cpuz80.ImmediateAddressing,
				Instruction:    cpuz80.LdImm8,
				RegisterParams: []cpuz80.RegisterParam{cpuz80.RegA},
				OperandValues:  []ast.Node{ast.NewNumber(42)},
			},
			want: []byte{0x3E, 0x2A},
		},
		{
			name:    "jr label",
			address: 0x1000,
			resolved: z80parser.ResolvedInstruction{
				Addressing:    cpuz80.RelativeAddressing,
				Instruction:   cpuz80.JrRel,
				OperandValues: []ast.Node{ast.NewLabel("loop")},
			},
			values: map[string]uint64{"loop": 0x1010},
			want:   []byte{0x18, 0x0E},
		},
		{
			name:    "jr nz,label",
			address: 0x1000,
			resolved: z80parser.ResolvedInstruction{
				Addressing:     cpuz80.RelativeAddressing,
				Instruction:    cpuz80.JrCond,
				RegisterParams: []cpuz80.RegisterParam{cpuz80.RegCondNZ},
				OperandValues:  []ast.Node{ast.NewLabel("loop")},
			},
			values: map[string]uint64{"loop": 0x1010},
			want:   []byte{0x20, 0x0E},
		},
		{
			name:    "bit 3,a",
			address: 0x8000,
			resolved: z80parser.ResolvedInstruction{
				Addressing:     cpuz80.RegisterAddressing,
				Instruction:    cpuz80.CBBit,
				RegisterParams: []cpuz80.RegisterParam{cpuz80.RegA},
				OperandValues:  []ast.Node{ast.NewNumber(3)},
			},
			want: []byte{0xCB, 0x5F},
		},
		{
			name:    "ld ix,nn",
			address: 0x8000,
			resolved: z80parser.ResolvedInstruction{
				Addressing:     cpuz80.ImmediateAddressing,
				Instruction:    cpuz80.DdLdIXnn,
				RegisterParams: []cpuz80.RegisterParam{cpuz80.RegIX},
				OperandValues:  []ast.Node{ast.NewNumber(0x1234)},
			},
			want: []byte{0xDD, 0x21, 0x34, 0x12},
		},
		{
			name:    "ddcb bit 3,(ix+5)",
			address: 0x8000,
			resolved: z80parser.ResolvedInstruction{
				Addressing:     cpuz80.BitAddressing,
				Instruction:    cpuz80.DdcbBit,
				RegisterParams: []cpuz80.RegisterParam{cpuz80.RegHLIndirect},
				OperandValues:  []ast.Node{ast.NewNumber(3), ast.NewNumber(5)},
			},
			want: []byte{0xDD, 0xCB, 0x05, 0x5E},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assigner := &mockAssigner{
				pc:     tt.address,
				values: tt.values,
			}
			ins := &mockInstruction{
				name:     tt.resolved.Instruction.Name,
				address:  tt.address,
				argument: tt.resolved,
			}

			err := GenerateInstructionOpcode(assigner, ins)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, ins.Opcodes())
			assert.Equal(t, len(tt.want), ins.Size())
		})
	}
}

func TestGenerateInstructionOpcode_Errors(t *testing.T) { //nolint:funlen
	tests := []struct {
		name     string
		address  uint64
		resolved z80parser.ResolvedInstruction
		wantErr  string
	}{
		{
			name: "relative out of range",
			resolved: z80parser.ResolvedInstruction{
				Addressing:    cpuz80.RelativeAddressing,
				Instruction:   cpuz80.JrRel,
				OperandValues: []ast.Node{ast.NewNumber(0xFFFF)},
			},
			wantErr: "relative offset",
		},
		{
			name: "missing operand for immediate",
			resolved: z80parser.ResolvedInstruction{
				Addressing:  cpuz80.ImmediateAddressing,
				Instruction: cpuz80.LdImm8,
			},
			wantErr: "missing operand value",
		},
		{
			name: "invalid bit number",
			resolved: z80parser.ResolvedInstruction{
				Addressing:     cpuz80.RegisterAddressing,
				Instruction:    cpuz80.CBBit,
				RegisterParams: []cpuz80.RegisterParam{cpuz80.RegA},
				OperandValues:  []ast.Node{ast.NewNumber(8)},
			},
			wantErr: "invalid bit number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assigner := &mockAssigner{
				pc: tt.address,
			}
			ins := &mockInstruction{
				name:     "test",
				address:  tt.address,
				argument: tt.resolved,
			}

			err := GenerateInstructionOpcode(assigner, ins)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
