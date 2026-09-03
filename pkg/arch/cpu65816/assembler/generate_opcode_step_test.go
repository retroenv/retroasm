package assembler

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/cpu65816/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu65816"
	"github.com/retroenv/retrogolib/assert"
)

func TestGenerateInstructionOpcode_ByteAddressing(t *testing.T) {
	tests := []struct {
		name       string
		insName    string
		addressing cpu65816.AddressingMode
		value      uint64
		wantErr    bool
	}{
		{"DirectPage valid", "lda", cpu65816.DirectPageAddressing, 0x10, false},
		{"DirectPage exceeds byte", "lda", cpu65816.DirectPageAddressing, 256, true},
		{"Immediate valid", "lda", cpu65816.ImmediateAddressing, 0x42, false},
		{"StackRelative valid", "lda", cpu65816.StackRelativeAddressing, 0x05, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assigner := &mockAssigner{}
			ins := &mockInstruction{
				name:       tt.insName,
				addressing: int(tt.addressing),
				argument:   testResolvedInstruction(tt.insName, tt.addressing, tt.value),
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
	assigner := &mockAssigner{}
	ins := &mockInstruction{
		name:       "jml",
		addressing: int(cpu65816.AbsoluteLongAddressing),
		argument:   testResolvedInstruction("jml", cpu65816.AbsoluteLongAddressing, 0x012345),
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
	assigner := &mockAssigner{}
	ins := &mockInstruction{
		name:       "brl",
		addressing: int(cpu65816.RelativeLongAddressing),
		argument:   testResolvedInstruction("brl", cpu65816.RelativeLongAddressing, 0x0100),
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
	// Encoding: opcode, dst(0x02), src(0x01)
	assigner := &mockAssigner{}
	ins := &mockInstruction{
		name:       "mvn",
		addressing: int(cpu65816.BlockMoveAddressing),
		argument: parser.ResolvedInstruction{
			Instruction: cpu65816.MvnInst,
			Addressing:  cpu65816.BlockMoveAddressing,
			Operands:    parser.BlockMoveOperands(ast.NewNumber(1), ast.NewNumber(2)),
			State:       parser.DefaultState(),
		},
	}

	err := GenerateInstructionOpcode(assigner, ins)
	assert.NoError(t, err)
	assert.Len(t, ins.opcodes, 3)
	assert.Equal(t, byte(0x54), ins.opcodes[0]) // MVN opcode
	assert.Equal(t, byte(0x02), ins.opcodes[1]) // dst bank
	assert.Equal(t, byte(0x01), ins.opcodes[2]) // src bank
	assert.Len(t, assigner.relocations, 2)
	assert.Equal(t, cpu65816Relocation(ast.AbsoluteRelocation, ast.WidthByte, 1), assigner.relocations[0].encoding)
	assert.Equal(t, cpu65816Relocation(ast.AbsoluteRelocation, ast.WidthByte, 2), assigner.relocations[1].encoding)
	destination := assigner.relocations[0].argument.(ast.InstructionReference)
	source := assigner.relocations[1].argument.(ast.InstructionReference)
	destinationValue, _ := ast.NumberValue(destination.Value)
	sourceValue, _ := ast.NumberValue(source.Value)
	assert.Equal(t, uint64(2), destinationValue)
	assert.Equal(t, uint64(1), sourceValue)
}

//nolint:funlen // The table keeps all encoded width families in one audit point.
func TestGenerateInstructionOpcode_RecordsRelocationEncoding(t *testing.T) {
	t.Parallel()

	wordState := parser.DefaultState()
	wordState.AccumulatorWidth = parser.WidthWord
	tests := []struct {
		name       string
		mnemonic   string
		addressing cpu65816.AddressingMode
		state      parser.State
		kind       ast.RelocationKind
		width      ast.DataWidth
	}{
		{name: "immediate byte", mnemonic: cpu65816.LdaName, addressing: cpu65816.ImmediateAddressing, state: parser.DefaultState(), kind: ast.AbsoluteRelocation, width: ast.WidthByte},
		{name: "immediate word", mnemonic: cpu65816.LdaName, addressing: cpu65816.ImmediateAddressing, state: wordState, kind: ast.AbsoluteRelocation, width: ast.WidthWord},
		{name: "direct page", mnemonic: cpu65816.LdaName, addressing: cpu65816.DirectPageAddressing, state: parser.DefaultState(), kind: ast.AbsoluteRelocation, width: ast.WidthByte},
		{name: "absolute", mnemonic: cpu65816.JmpName, addressing: cpu65816.AbsoluteAddressing, state: parser.DefaultState(), kind: ast.AbsoluteRelocation, width: ast.WidthWord},
		{name: "absolute long", mnemonic: cpu65816.JmlName, addressing: cpu65816.AbsoluteLongAddressing, state: parser.DefaultState(), kind: ast.AbsoluteRelocation, width: ast.WidthLong},
		{name: "relative", mnemonic: cpu65816.BneName, addressing: cpu65816.RelativeAddressing, state: parser.DefaultState(), kind: ast.RelativeRelocation, width: ast.WidthByte},
		{name: "relative long", mnemonic: cpu65816.BrlName, addressing: cpu65816.RelativeLongAddressing, state: parser.DefaultState(), kind: ast.RelativeRelocation, width: ast.WidthWord},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			resolved := testResolvedInstructionWithValue(
				test.mnemonic,
				test.addressing,
				ast.NewLabel("target"),
				test.state,
			)
			assigner := &mockAssigner{}
			ins := &mockInstruction{name: test.mnemonic, addressing: int(test.addressing), argument: resolved}
			err := GenerateInstructionOpcode(assigner, ins)
			assert.NoError(t, err)
			assert.Len(t, assigner.relocations, 1)
			assert.Equal(t, cpu65816Relocation(test.kind, test.width, 1), assigner.relocations[0].encoding)
			reference := assigner.relocations[0].argument.(ast.InstructionReference)
			assert.Equal(t, "target", ast.SymbolName(reference.Value))
		})
	}
}

func cpu65816Relocation(kind ast.RelocationKind, width ast.DataWidth, byteOffset uint64) arch.RelocationEncoding {
	return arch.RelocationEncoding{
		ByteOffset:    byteOffset,
		Kind:          kind,
		Width:         width,
		ByteOrder:     ast.ByteOrderLittle,
		ReferenceType: ast.FullAddress,
	}
}

type mockAssigner struct {
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

func (m *mockAssigner) ArgumentValue(argument any) (uint64, error) {
	value, _ := ast.NumberValue(argument.(ast.Node))
	return value, nil
}
func (m *mockAssigner) RelativeOffset(_, _ uint64) (byte, error) { return 0, nil }
func (m *mockAssigner) ProgramCounter() uint64                   { return 0 }
func (m *mockAssigner) RecordInstructionRelocation(_ arch.Instruction, argument any, encoding arch.RelocationEncoding) {
	m.relocations = append(m.relocations, recordedRelocation{argument: argument, encoding: encoding})
}

func (m *mockInstruction) Address() uint64        { return m.address }
func (m *mockInstruction) Addressing() int        { return m.addressing }
func (m *mockInstruction) Argument() any          { return m.argument }
func (m *mockInstruction) Name() string           { return m.name }
func (m *mockInstruction) Opcodes() []byte        { return m.opcodes }
func (m *mockInstruction) Size() int              { return m.size }
func (m *mockInstruction) OpcodeID() ast.OpcodeID { return ast.OpcodeID{} }
func (m *mockInstruction) SetAddress(a uint64)    { m.address = a }
func (m *mockInstruction) SetAddressing(a int)    { m.addressing = a }
func (m *mockInstruction) SetOpcodes(o []byte)    { m.opcodes = o }
func (m *mockInstruction) SetSize(s int)          { m.size = s }

func testResolvedInstruction(
	name string,
	addressing cpu65816.AddressingMode,
	value uint64,
) parser.ResolvedInstruction {

	return testResolvedInstructionWithValue(name, addressing, ast.NewNumber(value), parser.DefaultState())
}

func testResolvedInstructionWithValue(
	name string,
	addressing cpu65816.AddressingMode,
	value ast.Node,
	state parser.State,
) parser.ResolvedInstruction {

	return parser.ResolvedInstruction{
		Instruction: cpu65816.Instructions[name],
		Addressing:  addressing,
		Operands: parser.Operands{
			{Kind: parser.OperandAddress, Value: value},
		},
		State: state,
	}
}
