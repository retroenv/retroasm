package assembler

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu6502"
	"github.com/retroenv/retrogolib/assert"
)

func TestGenerateInstructionOpcode_IndirectXY(t *testing.T) {
	tests := []struct {
		name       string
		addressing cpu6502.AddressingMode
		value      uint64
		wantErr    bool
	}{
		{"IndirectX valid", cpu6502.IndirectXAddressing, 0x10, false},
		{"IndirectY valid", cpu6502.IndirectYAddressing, 0x80, false},
		{"IndirectX exceeds byte", cpu6502.IndirectXAddressing, 256, true},
		{"IndirectY exceeds byte", cpu6502.IndirectYAddressing, 300, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assigner := &mockAssigner{value: tt.value}
			ins := &mockInstruction{
				name:       "lda",
				addressing: int(tt.addressing),
				argument:   tt.value,
			}

			err := GenerateInstructionOpcode(assigner, ins, cpu6502.LdaInst)
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

func TestGenerateInstructionOpcode_RecordsRelocationEncoding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		addressing         cpu6502.AddressingMode
		value              uint64
		instruction        *cpu6502.Instruction
		expectedAddressing cpu6502.AddressingMode
		expected           arch.RelocationEncoding
	}{
		{
			name:               "zero page",
			addressing:         cpu6502.ZeroPageAddressing,
			value:              0x10,
			instruction:        cpu6502.LdaInst,
			expectedAddressing: cpu6502.ZeroPageAddressing,
			expected:           cpu6502Relocation(ast.AbsoluteRelocation, ast.WidthByte, 1),
		},
		{
			name:               "zero page upgraded to absolute",
			addressing:         cpu6502.ZeroPageAddressing,
			value:              0x100,
			instruction:        cpu6502.LdaInst,
			expectedAddressing: cpu6502.AbsoluteAddressing,
			expected:           cpu6502Relocation(ast.AbsoluteRelocation, ast.WidthWord, 1),
		},
		{
			name:               "absolute",
			addressing:         cpu6502.AbsoluteAddressing,
			value:              0x10,
			instruction:        cpu6502.JmpInst,
			expectedAddressing: cpu6502.AbsoluteAddressing,
			expected:           cpu6502Relocation(ast.AbsoluteRelocation, ast.WidthWord, 1),
		},
		{
			name:               "relative",
			addressing:         cpu6502.RelativeAddressing,
			value:              0x10,
			instruction:        cpu6502.BneInst,
			expectedAddressing: cpu6502.RelativeAddressing,
			expected:           cpu6502Relocation(ast.RelativeRelocation, ast.WidthByte, 1),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assigner := &mockAssigner{value: test.value}
			ins := &mockInstruction{name: test.instruction.Name, addressing: int(test.addressing), argument: "target"}
			err := GenerateInstructionOpcode(assigner, ins, test.instruction)
			assert.NoError(t, err)
			assert.Equal(t, int(test.expectedAddressing), ins.addressing)
			assert.Equal(t, []recordedRelocation{{argument: "target", encoding: test.expected}}, assigner.relocations)
		})
	}
}

func TestGenerateInstructionOpcode_RecordsZeroPageRelativeRelocations(t *testing.T) {
	t.Parallel()

	assigner := &mockAssigner{value: 0x10}
	ins := &mockInstruction{
		name:       cpu6502.Bbr0.Name,
		addressing: int(cpu6502.ZeroPageRelativeAddressing),
		argument:   []any{"zero-page", "target"},
	}
	err := GenerateInstructionOpcode(assigner, ins, cpu6502.Bbr0)
	assert.NoError(t, err)
	assert.Equal(t, []recordedRelocation{
		{argument: "zero-page", encoding: cpu6502Relocation(ast.AbsoluteRelocation, ast.WidthByte, 1)},
		{argument: "target", encoding: cpu6502Relocation(ast.RelativeRelocation, ast.WidthByte, 2)},
	}, assigner.relocations)
}

func cpu6502Relocation(kind ast.RelocationKind, width ast.DataWidth, byteOffset uint64) arch.RelocationEncoding {
	return arch.RelocationEncoding{
		ByteOffset:    byteOffset,
		Kind:          kind,
		Width:         width,
		ByteOrder:     ast.ByteOrderLittle,
		ReferenceType: ast.FullAddress,
	}
}

type mockAssigner struct {
	value       uint64
	relocations []recordedRelocation
}

type recordedRelocation struct {
	argument any
	encoding arch.RelocationEncoding
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
func (m *mockAssigner) RecordInstructionRelocation(_ arch.Instruction, argument any, encoding arch.RelocationEncoding) {
	m.relocations = append(m.relocations, recordedRelocation{argument: argument, encoding: encoding})
}

func (m *mockInstruction) Address() uint64        { return m.address }
func (m *mockInstruction) Addressing() int        { return m.addressing }
func (m *mockInstruction) Argument() any          { return m.argument }
func (m *mockInstruction) Name() string           { return m.name }
func (m *mockInstruction) OpcodeID() ast.OpcodeID { return ast.OpcodeID{} }
func (m *mockInstruction) Opcodes() []byte        { return m.opcodes }
func (m *mockInstruction) Size() int              { return m.size }
func (m *mockInstruction) SetAddress(a uint64)    { m.address = a }
func (m *mockInstruction) SetAddressing(a int)    { m.addressing = a }
func (m *mockInstruction) SetOpcodes(o []byte)    { m.opcodes = o }
func (m *mockInstruction) SetSize(s int)          { m.size = s }
