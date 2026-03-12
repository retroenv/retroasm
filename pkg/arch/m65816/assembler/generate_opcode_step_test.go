package assembler

import (
	"testing"

	"github.com/retroenv/retrogolib/arch/cpu/m65816"
	"github.com/retroenv/retrogolib/assert"
)

func TestGenerateInstructionOpcode_ByteAddressing(t *testing.T) {
	tests := []struct {
		name       string
		insName    string
		addressing m65816.AddressingMode
		value      uint64
		wantErr    bool
	}{
		{"DirectPage valid", "lda", m65816.DirectPageAddressing, 0x10, false},
		{"DirectPage exceeds byte", "lda", m65816.DirectPageAddressing, 256, true},
		{"Immediate valid", "lda", m65816.ImmediateAddressing, 0x42, false},
		{"StackRelative valid", "lda", m65816.StackRelativeAddressing, 0x05, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assigner := &mockAssigner{value: tt.value}
			ins := &mockInstruction{
				name:       tt.insName,
				addressing: int(tt.addressing),
				argument:   tt.value,
			}

			err := GenerateInstructionOpcode(assigner, ins)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, ins.opcodes, 2)
			assert.Equal(t, byte(tt.value), ins.opcodes[1])
		})
	}
}

func TestGenerateInstructionOpcode_LongAddress(t *testing.T) {
	assigner := &mockAssigner{value: 0x012345}
	ins := &mockInstruction{
		name:       "jml",
		addressing: int(m65816.AbsoluteLongAddressing),
		argument:   uint64(0x012345),
	}

	err := GenerateInstructionOpcode(assigner, ins)
	assert.NoError(t, err)
	assert.Len(t, ins.opcodes, 4)
	assert.Equal(t, byte(0x5C), ins.opcodes[0]) // JML opcode
	assert.Equal(t, byte(0x45), ins.opcodes[1]) // low byte
	assert.Equal(t, byte(0x23), ins.opcodes[2]) // mid byte
	assert.Equal(t, byte(0x01), ins.opcodes[3]) // bank byte
}

func TestGenerateInstructionOpcode_RelativeLong(t *testing.T) {
	// BRL from address 0x0000 (after instruction = 0x0003) to target 0x0100
	// offset = 0x0100 - 0x0003 = 0x00FD
	assigner := &mockAssigner{value: 0x0100}
	ins := &mockInstruction{
		name:       "brl",
		addressing: int(m65816.RelativeLongAddressing),
		argument:   uint64(0x0100),
		address:    0x0000,
		size:       3,
	}

	err := GenerateInstructionOpcode(assigner, ins)
	assert.NoError(t, err)
	assert.Len(t, ins.opcodes, 3)
	assert.Equal(t, byte(0x82), ins.opcodes[0]) // BRL opcode
	assert.Equal(t, byte(0xFD), ins.opcodes[1]) // low byte of offset
	assert.Equal(t, byte(0x00), ins.opcodes[2]) // high byte of offset
}

func TestGenerateInstructionOpcode_BlockMove(t *testing.T) {
	// MVN $01,$02 → packed as (0x01 << 8) | 0x02 = 0x0102
	// Encoding: opcode, dst(0x02), src(0x01)
	assigner := &mockAssigner{value: 0x0102}
	ins := &mockInstruction{
		name:       "mvn",
		addressing: int(m65816.BlockMoveAddressing),
		argument:   uint64(0x0102),
	}

	err := GenerateInstructionOpcode(assigner, ins)
	assert.NoError(t, err)
	assert.Len(t, ins.opcodes, 3)
	assert.Equal(t, byte(0x54), ins.opcodes[0]) // MVN opcode
	assert.Equal(t, byte(0x02), ins.opcodes[1]) // dst bank
	assert.Equal(t, byte(0x01), ins.opcodes[2]) // src bank
}

type mockAssigner struct {
	value uint64
}

type mockInstruction struct {
	name       string
	addressing int
	argument   any
	opcodes    []byte
	size       int
	address    uint64
}

func (m *mockAssigner) ArgumentValue(_ any) (uint64, error)      { return m.value, nil }
func (m *mockAssigner) RelativeOffset(_, _ uint64) (byte, error) { return 0, nil }
func (m *mockAssigner) ProgramCounter() uint64                   { return 0 }

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
