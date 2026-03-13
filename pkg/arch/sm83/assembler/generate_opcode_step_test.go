package assembler

import (
	"errors"
	"fmt"
	"testing"

	"github.com/retroenv/retroasm/pkg/arch"
	sm83parser "github.com/retroenv/retroasm/pkg/arch/sm83/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpusm83 "github.com/retroenv/retrogolib/arch/cpu/sm83"
	"github.com/retroenv/retrogolib/assert"
)

var coreEncodingTests = []struct {
	name     string
	address  uint64
	resolved sm83parser.ResolvedInstruction
	values   map[string]uint64
	want     []byte
}{
	{
		name:    "nop",
		address: 0x0000,
		resolved: sm83parser.ResolvedInstruction{
			Addressing:  cpusm83.ImpliedAddressing,
			Instruction: cpusm83.Nop,
		},
		want: []byte{0x00},
	},
	{
		name:    "ld bc,nn",
		address: 0x0000,
		resolved: sm83parser.ResolvedInstruction{
			Addressing:     cpusm83.ImmediateAddressing,
			Instruction:    cpusm83.LdReg16,
			RegisterParams: []cpusm83.RegisterParam{cpusm83.RegBC},
			OperandValues:  []ast.Node{ast.NewNumber(0x1234)},
		},
		want: []byte{0x01, 0x34, 0x12},
	},
	{
		name:    "ld a,n",
		address: 0x0000,
		resolved: sm83parser.ResolvedInstruction{
			Addressing:     cpusm83.ImmediateAddressing,
			Instruction:    cpusm83.LdImm8,
			RegisterParams: []cpusm83.RegisterParam{cpusm83.RegA},
			OperandValues:  []ast.Node{ast.NewNumber(0x42)},
		},
		want: []byte{0x3E, 0x42},
	},
	{
		name:    "jr label",
		address: 0x1000,
		resolved: sm83parser.ResolvedInstruction{
			Addressing:    cpusm83.RelativeAddressing,
			Instruction:   cpusm83.JrRel,
			OperandValues: []ast.Node{ast.NewLabel("loop")},
		},
		values: map[string]uint64{"loop": 0x1010},
		want:   []byte{0x18, 0x0E},
	},
	{
		name:    "jr nz,label",
		address: 0x1000,
		resolved: sm83parser.ResolvedInstruction{
			Addressing:     cpusm83.RelativeAddressing,
			Instruction:    cpusm83.JrCond,
			RegisterParams: []cpusm83.RegisterParam{cpusm83.RegCondNZ},
			OperandValues:  []ast.Node{ast.NewLabel("loop")},
		},
		values: map[string]uint64{"loop": 0x1010},
		want:   []byte{0x20, 0x0E},
	},
	{
		name:    "bit 3,a",
		address: 0x0000,
		resolved: sm83parser.ResolvedInstruction{
			Addressing:     cpusm83.BitAddressing,
			Instruction:    cpusm83.CBBit,
			RegisterParams: []cpusm83.RegisterParam{cpusm83.RegA},
			OperandValues:  []ast.Node{ast.NewNumber(3)},
		},
		want: []byte{0xCB, 0x5F},
	},
	{
		name:    "swap a (cb prefix)",
		address: 0x0000,
		resolved: sm83parser.ResolvedInstruction{
			Addressing:     cpusm83.RegisterAddressing,
			Instruction:    cpusm83.CBSwap,
			RegisterParams: []cpusm83.RegisterParam{cpusm83.RegA},
		},
		want: []byte{0xCB, 0x37},
	},
	{
		name:    "rlc b (cb prefix)",
		address: 0x0000,
		resolved: sm83parser.ResolvedInstruction{
			Addressing:     cpusm83.RegisterAddressing,
			Instruction:    cpusm83.CBRlc,
			RegisterParams: []cpusm83.RegisterParam{cpusm83.RegB},
		},
		want: []byte{0xCB, 0x00},
	},
	{
		name:    "jp nn",
		address: 0x0000,
		resolved: sm83parser.ResolvedInstruction{
			Addressing:    cpusm83.ExtendedAddressing,
			Instruction:   cpusm83.JpAbs,
			OperandValues: []ast.Node{ast.NewNumber(0x8000)},
		},
		want: []byte{0xC3, 0x00, 0x80},
	},
	{
		name:    "push bc",
		address: 0x0000,
		resolved: sm83parser.ResolvedInstruction{
			Addressing:     cpusm83.RegisterAddressing,
			Instruction:    cpusm83.PushReg16,
			RegisterParams: []cpusm83.RegisterParam{cpusm83.RegBC},
		},
		want: []byte{0xC5},
	},
	{
		name:    "rst 38h",
		address: 0x0000,
		resolved: sm83parser.ResolvedInstruction{
			Addressing:     cpusm83.ImpliedAddressing,
			Instruction:    cpusm83.Rst,
			RegisterParams: []cpusm83.RegisterParam{cpusm83.RegRst38},
		},
		want: []byte{0xFF},
	},
	{
		name:    "inc b",
		address: 0x0000,
		resolved: sm83parser.ResolvedInstruction{
			Addressing:     cpusm83.RegisterAddressing,
			Instruction:    cpusm83.IncReg8,
			RegisterParams: []cpusm83.RegisterParam{cpusm83.RegB},
		},
		want: []byte{0x04},
	},
	{
		name:    "ld a,b",
		address: 0x0000,
		resolved: sm83parser.ResolvedInstruction{
			Addressing:     cpusm83.RegisterAddressing,
			Instruction:    cpusm83.LdReg8,
			RegisterParams: []cpusm83.RegisterParam{cpusm83.RegA, cpusm83.RegB},
		},
		want: []byte{0x78},
	},
}

func TestGenerateInstructionOpcode_CoreEncodings(t *testing.T) {
	for _, tt := range coreEncodingTests {
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

func TestGenerateInstructionOpcode_Errors(t *testing.T) {
	tests := []struct {
		name     string
		resolved sm83parser.ResolvedInstruction
		wantErr  string
	}{
		{
			name: "missing operand for immediate",
			resolved: sm83parser.ResolvedInstruction{
				Addressing:  cpusm83.ImmediateAddressing,
				Instruction: cpusm83.LdImm8,
			},
			wantErr: "missing operand value",
		},
		{
			name: "invalid bit number",
			resolved: sm83parser.ResolvedInstruction{
				Addressing:     cpusm83.BitAddressing,
				Instruction:    cpusm83.CBBit,
				RegisterParams: []cpusm83.RegisterParam{cpusm83.RegA},
				OperandValues:  []ast.Node{ast.NewNumber(8)},
			},
			wantErr: "invalid bit number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assigner := &mockAssigner{pc: 0x0000}
			ins := &mockInstruction{
				name:     "test",
				address:  0x0000,
				argument: tt.resolved,
			}

			err := GenerateInstructionOpcode(assigner, ins)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

var boundaryTests = []struct {
	name     string
	address  uint64
	resolved sm83parser.ResolvedInstruction
	want     []byte
}{
	{
		name:    "jp nn minimum address",
		address: 0x0000,
		resolved: sm83parser.ResolvedInstruction{
			Addressing:    cpusm83.ExtendedAddressing,
			Instruction:   cpusm83.JpAbs,
			OperandValues: []ast.Node{ast.NewNumber(0x0000)},
		},
		want: []byte{0xC3, 0x00, 0x00},
	},
	{
		name:    "jp nn maximum address",
		address: 0x0000,
		resolved: sm83parser.ResolvedInstruction{
			Addressing:    cpusm83.ExtendedAddressing,
			Instruction:   cpusm83.JpAbs,
			OperandValues: []ast.Node{ast.NewNumber(0xFFFF)},
		},
		want: []byte{0xC3, 0xFF, 0xFF},
	},
	{
		name:    "jr relative positive boundary +127",
		address: 0x1000,
		resolved: sm83parser.ResolvedInstruction{
			Addressing:    cpusm83.RelativeAddressing,
			Instruction:   cpusm83.JrRel,
			OperandValues: []ast.Node{ast.NewNumber(0x1081)},
		},
		want: []byte{0x18, 0x7F},
	},
	{
		name:    "jr relative negative boundary -128",
		address: 0x1000,
		resolved: sm83parser.ResolvedInstruction{
			Addressing:    cpusm83.RelativeAddressing,
			Instruction:   cpusm83.JrRel,
			OperandValues: []ast.Node{ast.NewNumber(0x0F82)},
		},
		want: []byte{0x18, 0x80},
	},
	{
		name:    "set 7,a",
		address: 0x0000,
		resolved: sm83parser.ResolvedInstruction{
			Addressing:     cpusm83.BitAddressing,
			Instruction:    cpusm83.CBSet,
			RegisterParams: []cpusm83.RegisterParam{cpusm83.RegA},
			OperandValues:  []ast.Node{ast.NewNumber(7)},
		},
		want: []byte{0xCB, 0xFF},
	},
	{
		name:    "res 0,(hl)",
		address: 0x0000,
		resolved: sm83parser.ResolvedInstruction{
			Addressing:     cpusm83.BitAddressing,
			Instruction:    cpusm83.CBRes,
			RegisterParams: []cpusm83.RegisterParam{cpusm83.RegHLIndirect},
			OperandValues:  []ast.Node{ast.NewNumber(0)},
		},
		want: []byte{0xCB, 0x86},
	},
}

func TestGenerateInstructionOpcode_Boundaries(t *testing.T) {
	for _, tt := range boundaryTests {
		t.Run(tt.name, func(t *testing.T) {
			assigner := &mockAssigner{pc: tt.address}
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

func TestAssignInstructionAddress_SetsAddressingAndSize(t *testing.T) {
	tests := []struct {
		name           string
		resolved       sm83parser.ResolvedInstruction
		wantSize       int
		wantAddressing cpusm83.AddressingMode
	}{
		{
			name: "one byte instruction nop",
			resolved: sm83parser.ResolvedInstruction{
				Addressing:  cpusm83.ImpliedAddressing,
				Instruction: cpusm83.Nop,
			},
			wantSize:       1,
			wantAddressing: cpusm83.ImpliedAddressing,
		},
		{
			name: "two byte instruction ld a,n",
			resolved: sm83parser.ResolvedInstruction{
				Addressing:     cpusm83.ImmediateAddressing,
				Instruction:    cpusm83.LdImm8,
				RegisterParams: []cpusm83.RegisterParam{cpusm83.RegA},
			},
			wantSize:       2,
			wantAddressing: cpusm83.ImmediateAddressing,
		},
		{
			name: "three byte instruction ld hl,nn",
			resolved: sm83parser.ResolvedInstruction{
				Addressing:     cpusm83.ImmediateAddressing,
				Instruction:    cpusm83.LdReg16,
				RegisterParams: []cpusm83.RegisterParam{cpusm83.RegHL},
			},
			wantSize:       3,
			wantAddressing: cpusm83.ImmediateAddressing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assigner := &mockAssigner{pc: 0x0000}
			ins := &mockInstruction{
				name:     tt.resolved.Instruction.Name,
				argument: tt.resolved,
			}

			nextPC, err := AssignInstructionAddress(assigner, ins)
			assert.NoError(t, err)
			assert.Equal(t, uint64(0x0000), ins.Address())
			assert.Equal(t, int(tt.wantAddressing), ins.Addressing())
			assert.Equal(t, tt.wantSize, ins.Size())
			assert.Equal(t, uint64(tt.wantSize), nextPC)
		})
	}
}

func TestAssignInstructionAddress_Errors(t *testing.T) {
	tests := []struct {
		name     string
		argument any
		wantErr  error
	}{
		{
			name:     "unsupported argument type",
			argument: "invalid",
			wantErr:  errUnsupportedArgumentType,
		},
		{
			name:     "missing instruction details",
			argument: sm83parser.ResolvedInstruction{},
			wantErr:  errMissingInstruction,
		},
		{
			name: "opcode not found",
			argument: sm83parser.ResolvedInstruction{
				Addressing:     cpusm83.ImmediateAddressing,
				Instruction:    cpusm83.Nop,
				RegisterParams: []cpusm83.RegisterParam{cpusm83.RegA},
			},
			wantErr: errOpcodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assigner := &mockAssigner{pc: 0x0000}
			ins := &mockInstruction{
				name:     "test",
				argument: tt.argument,
			}

			_, err := AssignInstructionAddress(assigner, ins)
			assert.Error(t, err)
			assert.True(t, errors.Is(err, tt.wantErr))
		})
	}
}

type mockAssigner struct {
	pc     uint64
	values map[string]uint64
}

type mockInstruction struct {
	name       string
	addressing int
	argument   any
	opcodes    []byte
	size       int
	address    uint64
}

func (m *mockAssigner) ArgumentValue(argument any) (uint64, error) {
	switch value := argument.(type) {
	case uint64:
		return value, nil
	case int:
		return uint64(value), nil
	case ast.Number:
		return value.Value, nil
	case ast.Label:
		if m.values == nil {
			return 0, fmt.Errorf("value for label '%s' not configured", value.Name)
		}
		resolved, ok := m.values[value.Name]
		if !ok {
			return 0, fmt.Errorf("value for label '%s' not configured", value.Name)
		}
		return resolved, nil
	default:
		return 0, fmt.Errorf("unsupported argument type %T", argument)
	}
}

func (m *mockAssigner) RelativeOffset(destination, addressAfterInstruction uint64) (byte, error) {
	diff := int64(destination) - int64(addressAfterInstruction)
	switch {
	case diff < -128 || diff > 127:
		return 0, fmt.Errorf("relative distance %d exceeds limit", diff)
	case diff >= 0:
		return byte(diff), nil
	default:
		return byte(256 + diff), nil
	}
}

func (m *mockAssigner) ProgramCounter() uint64 { return m.pc }

func (m *mockInstruction) Address() uint64     { return m.address }
func (m *mockInstruction) Addressing() int     { return m.addressing }
func (m *mockInstruction) Argument() any       { return m.argument }
func (m *mockInstruction) Name() string        { return m.name }
func (m *mockInstruction) Opcodes() []byte     { return m.opcodes }
func (m *mockInstruction) Size() int           { return m.size }
func (m *mockInstruction) SetAddress(a uint64) { m.address = a }
func (m *mockInstruction) SetAddressing(a int) { m.addressing = a }
func (m *mockInstruction) SetOpcodes(o []byte) { m.opcodes = o }
func (m *mockInstruction) SetSize(s int)       { m.size = s }

var _ arch.AddressAssigner = (*mockAssigner)(nil)
var _ arch.Instruction = (*mockInstruction)(nil)
