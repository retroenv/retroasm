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
			assert.Len(t, tt.want, ins.Size())
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

func TestGenerateInstructionOpcode_BoundaryMatrix(t *testing.T) { //nolint:funlen
	tests := []struct {
		name     string
		address  uint64
		resolved z80parser.ResolvedInstruction
		want     []byte
	}{
		{
			name:    "jp nn minimum extended address",
			address: 0x8000,
			resolved: z80parser.ResolvedInstruction{
				Addressing:    cpuz80.ExtendedAddressing,
				Instruction:   cpuz80.JpAbs,
				OperandValues: []ast.Node{ast.NewNumber(0x0000)},
			},
			want: []byte{0xC3, 0x00, 0x00},
		},
		{
			name:    "jp nn maximum extended address",
			address: 0x8000,
			resolved: z80parser.ResolvedInstruction{
				Addressing:    cpuz80.ExtendedAddressing,
				Instruction:   cpuz80.JpAbs,
				OperandValues: []ast.Node{ast.NewNumber(0xFFFF)},
			},
			want: []byte{0xC3, 0xFF, 0xFF},
		},
		{
			name:    "in a,(n) minimum port",
			address: 0x8000,
			resolved: z80parser.ResolvedInstruction{
				Addressing:    cpuz80.PortAddressing,
				Instruction:   cpuz80.InPort,
				OperandValues: []ast.Node{ast.NewNumber(0x00)},
			},
			want: []byte{0xDB, 0x00},
		},
		{
			name:    "in a,(n) maximum port",
			address: 0x8000,
			resolved: z80parser.ResolvedInstruction{
				Addressing:    cpuz80.PortAddressing,
				Instruction:   cpuz80.InPort,
				OperandValues: []ast.Node{ast.NewNumber(0xFF)},
			},
			want: []byte{0xDB, 0xFF},
		},
		{
			name:    "out (n),a minimum port",
			address: 0x8000,
			resolved: z80parser.ResolvedInstruction{
				Addressing:    cpuz80.PortAddressing,
				Instruction:   cpuz80.OutPort,
				OperandValues: []ast.Node{ast.NewNumber(0x00)},
			},
			want: []byte{0xD3, 0x00},
		},
		{
			name:    "out (n),a maximum port",
			address: 0x8000,
			resolved: z80parser.ResolvedInstruction{
				Addressing:    cpuz80.PortAddressing,
				Instruction:   cpuz80.OutPort,
				OperandValues: []ast.Node{ast.NewNumber(0xFF)},
			},
			want: []byte{0xD3, 0xFF},
		},
		{
			name:    "ld a,(ix+0) minimum displacement",
			address: 0x8000,
			resolved: z80parser.ResolvedInstruction{
				Addressing:     cpuz80.RegisterIndirectAddressing,
				Instruction:    cpuz80.DdLdAIXd,
				RegisterParams: []cpuz80.RegisterParam{cpuz80.RegA},
				OperandValues:  []ast.Node{ast.NewNumber(0x00)},
			},
			want: []byte{0xDD, 0x7E, 0x00},
		},
		{
			name:    "ld a,(ix-1) maximum unsigned displacement byte",
			address: 0x8000,
			resolved: z80parser.ResolvedInstruction{
				Addressing:     cpuz80.RegisterIndirectAddressing,
				Instruction:    cpuz80.DdLdAIXd,
				RegisterParams: []cpuz80.RegisterParam{cpuz80.RegA},
				OperandValues:  []ast.Node{ast.NewNumber(0xFF)},
			},
			want: []byte{0xDD, 0x7E, 0xFF},
		},
		{
			name:    "jr relative positive boundary +127",
			address: 0x1000,
			resolved: z80parser.ResolvedInstruction{
				Addressing:    cpuz80.RelativeAddressing,
				Instruction:   cpuz80.JrRel,
				OperandValues: []ast.Node{ast.NewNumber(0x1081)},
			},
			want: []byte{0x18, 0x7F},
		},
		{
			name:    "jr relative negative boundary -128",
			address: 0x1000,
			resolved: z80parser.ResolvedInstruction{
				Addressing:    cpuz80.RelativeAddressing,
				Instruction:   cpuz80.JrRel,
				OperandValues: []ast.Node{ast.NewNumber(0x0F82)},
			},
			want: []byte{0x18, 0x80},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assigner := &mockAssigner{
				pc: tt.address,
			}
			ins := &mockInstruction{
				name:     "boundary",
				address:  tt.address,
				argument: tt.resolved,
			}

			err := GenerateInstructionOpcode(assigner, ins)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, ins.Opcodes())
			assert.Len(t, tt.want, ins.Size())
		})
	}
}
