package assembler

import (
	"errors"
	"testing"

	"github.com/retroenv/retroasm/pkg/arch"
	z80parser "github.com/retroenv/retroasm/pkg/arch/z80/parser"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
	"github.com/retroenv/retrogolib/assert"
)

type mockAssigner struct {
	pc uint64
}

func (m *mockAssigner) ArgumentValue(_ any) (uint64, error)      { return 0, nil }
func (m *mockAssigner) RelativeOffset(_, _ uint64) (byte, error) { return 0, nil }
func (m *mockAssigner) ProgramCounter() uint64                   { return m.pc }

type mockInstruction struct {
	name       string
	addressing int
	argument   any
	opcodes    []byte
	size       int
	address    uint64
}

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

func TestAssignInstructionAddress_SetsAddressingAndSize(t *testing.T) { //nolint:funlen
	tests := []struct {
		name           string
		resolved       z80parser.ResolvedInstruction
		wantSize       int
		wantAddressing cpuz80.AddressingMode
	}{
		{
			name: "one byte instruction nop",
			resolved: z80parser.ResolvedInstruction{
				Addressing:  cpuz80.ImpliedAddressing,
				Instruction: cpuz80.Nop,
			},
			wantSize:       1,
			wantAddressing: cpuz80.ImpliedAddressing,
		},
		{
			name: "two byte instruction ld a,n",
			resolved: z80parser.ResolvedInstruction{
				Addressing:     cpuz80.ImmediateAddressing,
				Instruction:    cpuz80.LdImm8,
				RegisterParams: []cpuz80.RegisterParam{cpuz80.RegA},
			},
			wantSize:       2,
			wantAddressing: cpuz80.ImmediateAddressing,
		},
		{
			name: "three byte instruction ld hl,nn",
			resolved: z80parser.ResolvedInstruction{
				Addressing:     cpuz80.ImmediateAddressing,
				Instruction:    cpuz80.LdReg16,
				RegisterParams: []cpuz80.RegisterParam{cpuz80.RegHL},
			},
			wantSize:       3,
			wantAddressing: cpuz80.ImmediateAddressing,
		},
		{
			name: "four byte instruction ld ix,nn",
			resolved: z80parser.ResolvedInstruction{
				Addressing:     cpuz80.ImmediateAddressing,
				Instruction:    cpuz80.DdLdIXnn,
				RegisterParams: []cpuz80.RegisterParam{cpuz80.RegIX},
			},
			wantSize:       4,
			wantAddressing: cpuz80.ImmediateAddressing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assigner := &mockAssigner{pc: 0x8000}
			ins := &mockInstruction{
				name:     tt.resolved.Instruction.Name,
				argument: tt.resolved,
			}

			nextPC, err := AssignInstructionAddress(assigner, ins)
			assert.NoError(t, err)
			assert.Equal(t, uint64(0x8000), ins.Address())
			assert.Equal(t, int(tt.wantAddressing), ins.Addressing())
			assert.Equal(t, tt.wantSize, ins.Size())
			assert.Equal(t, uint64(0x8000+tt.wantSize), nextPC)
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
			argument: z80parser.ResolvedInstruction{},
			wantErr:  errMissingInstruction,
		},
		{
			name: "opcode not found",
			argument: z80parser.ResolvedInstruction{
				Addressing:     cpuz80.ImmediateAddressing,
				Instruction:    cpuz80.Nop,
				RegisterParams: []cpuz80.RegisterParam{cpuz80.RegA},
			},
			wantErr: errOpcodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assigner := &mockAssigner{pc: 0x4000}
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

var _ arch.AddressAssigner = (*mockAssigner)(nil)
var _ arch.Instruction = (*mockInstruction)(nil)
