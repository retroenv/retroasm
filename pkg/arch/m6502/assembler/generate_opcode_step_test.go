package assembler

import (
	"testing"

	"github.com/retroenv/retrogolib/arch/cpu/m6502"
	"github.com/retroenv/retrogolib/assert"
)

type mockAssigner struct {
	value uint64
}

func (m *mockAssigner) ArgumentValue(_ any) (uint64, error)      { return m.value, nil }
func (m *mockAssigner) RelativeOffset(_, _ uint64) (byte, error) { return 0, nil }
func (m *mockAssigner) ProgramCounter() uint64                   { return 0 }

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

func TestGenerateInstructionOpcode_IndirectXY(t *testing.T) {
	tests := []struct {
		name       string
		addressing m6502.AddressingMode
		value      uint64
		wantErr    bool
	}{
		{"IndirectX valid", m6502.IndirectXAddressing, 0x10, false},
		{"IndirectY valid", m6502.IndirectYAddressing, 0x80, false},
		{"IndirectX exceeds byte", m6502.IndirectXAddressing, 256, true},
		{"IndirectY exceeds byte", m6502.IndirectYAddressing, 300, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assigner := &mockAssigner{value: tt.value}
			ins := &mockInstruction{
				name:       "lda",
				addressing: int(tt.addressing),
				argument:   tt.value,
			}

			err := GenerateInstructionOpcode(assigner, ins)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, 2, len(ins.opcodes))
			assert.Equal(t, byte(tt.value), ins.opcodes[1])
		})
	}
}
