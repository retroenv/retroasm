package assembler

import (
	"fmt"
	"testing"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/chip8/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/chip8"
	"github.com/retroenv/retrogolib/assert"
)

func TestGenerateInstructionOpcode_RecordsRelocations(t *testing.T) { //nolint:funlen // The table audits packed field layouts.
	tests := []struct {
		name        string
		instruction string
		addressing  chip8.Mode
		operands    parser.Operands
		value       uint64
		wantCode    []byte
		want        arch.RelocationEncoding
	}{
		{
			name: "absolute address", instruction: chip8.CallName, addressing: chip8.AbsoluteAddressing,
			operands: parser.Operands{parser.AddressOperand(ast.NewLabel("target"))}, value: 0x234,
			wantCode: []byte{0x22, 0x34},
			want:     chip8AddressRelocationEncoding(),
		},
		{
			name: "V0 absolute address", instruction: chip8.JpName, addressing: chip8.V0AbsoluteAddressing,
			operands: parser.Operands{parser.RegisterOperand(0), parser.AddressOperand(ast.NewLabel("target"))}, value: 0x345,
			wantCode: []byte{0xb3, 0x45},
			want:     chip8AddressRelocationEncoding(),
		},
		{
			name: "I absolute address", instruction: chip8.LdName, addressing: chip8.IAbsoluteAddressing,
			operands: parser.Operands{parser.SpecialOperand(parser.OperandI), parser.AddressOperand(ast.NewLabel("target"))}, value: 0x456,
			wantCode: []byte{0xa4, 0x56},
			want:     chip8AddressRelocationEncoding(),
		},
		{
			name: "register byte", instruction: chip8.LdName, addressing: chip8.RegisterValueAddressing,
			operands: parser.Operands{parser.RegisterOperand(3), parser.ByteOperand(ast.NewLabel("target"))}, value: 0x7f,
			wantCode: []byte{0x63, 0x7f},
			want: arch.RelocationEncoding{
				ByteOffset: 1, Kind: ast.AbsoluteRelocation, Width: ast.WidthByte,
				ByteOrder: ast.ByteOrderBig, ReferenceType: ast.FullAddress,
			},
		},
		{
			name: "register pair nibble", instruction: chip8.DrwName, addressing: chip8.RegisterRegisterNibbleAddressing,
			operands: parser.Operands{
				parser.RegisterOperand(1), parser.RegisterOperand(2), parser.NibbleOperand(ast.NewLabel("target")),
			},
			value: 5, wantCode: []byte{0xd1, 0x25},
			want: arch.RelocationEncoding{
				ByteOffset: 1, Kind: ast.AbsoluteRelocation, Width: ast.WidthByte,
				ByteOrder: ast.ByteOrderBig, ReferenceType: ast.FullAddress,
				Field: ast.PackedField{BitWidth: 4, PreserveMask: 0xf0},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolved := parser.ResolvedInstruction{
				Instruction: chip8.Instructions[test.instruction], Addressing: test.addressing, Operands: test.operands,
			}
			assigner := &relocationAssigner{values: map[string]uint64{"target": test.value}}
			instruction := &mockInstruction{name: test.instruction, addressing: int(test.addressing), argument: resolved}

			err := GenerateInstructionOpcode(assigner, instruction)
			assert.NoError(t, err)
			assert.Equal(t, test.wantCode, instruction.opcodes)
			assert.Len(t, assigner.relocations, 1)
			assert.Equal(t, test.want, assigner.relocations[0].encoding)
			assert.Equal(t, "target", ast.SymbolName(assigner.relocations[0].argument.(ast.Node)))
		})
	}
}

func TestGenerateInstructionOpcode_DoesNotRecordImpliedRelocation(t *testing.T) {
	resolved := parser.ResolvedInstruction{
		Instruction: chip8.Instructions[chip8.ClsName],
		Addressing:  chip8.ImpliedAddressing,
	}
	assigner := &relocationAssigner{}
	instruction := &mockInstruction{name: chip8.ClsName, addressing: int(chip8.ImpliedAddressing), argument: resolved}

	err := GenerateInstructionOpcode(assigner, instruction)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x00, 0xe0}, instruction.opcodes)
	assert.Empty(t, assigner.relocations)
}

func chip8AddressRelocationEncoding() arch.RelocationEncoding {
	return arch.RelocationEncoding{
		Kind: ast.AbsoluteRelocation, Width: ast.WidthWord,
		ByteOrder: ast.ByteOrderBig, ReferenceType: ast.FullAddress,
		Field: ast.PackedField{BitWidth: 12, PreserveMask: 0xf000},
	}
}

type recordedRelocation struct {
	argument any
	encoding arch.RelocationEncoding
}

type relocationAssigner struct {
	values      map[string]uint64
	relocations []recordedRelocation
}

func (ass *relocationAssigner) ArgumentValue(argument any) (uint64, error) {
	if value, ok := ast.NumberValue(argument.(ast.Node)); ok {
		return value, nil
	}
	symbol := ast.SymbolName(argument.(ast.Node))
	if value, ok := ass.values[symbol]; ok {
		return value, nil
	}
	return 0, fmt.Errorf("symbol %q not found", symbol)
}

func (*relocationAssigner) ProgramCounter() uint64 {
	return 0
}

func (*relocationAssigner) RelativeOffset(uint64, uint64) (byte, error) {
	return 0, nil
}

func (ass *relocationAssigner) RecordInstructionRelocation(
	_ arch.Instruction,
	argument any,
	encoding arch.RelocationEncoding,
) {

	ass.relocations = append(ass.relocations, recordedRelocation{argument: argument, encoding: encoding})
}

type mockInstruction struct {
	name       string
	addressing int
	argument   any
	opcodes    []byte
	size       int
	address    uint64
}

func (ins *mockInstruction) Address() uint64        { return ins.address }
func (ins *mockInstruction) Addressing() int        { return ins.addressing }
func (ins *mockInstruction) Argument() any          { return ins.argument }
func (ins *mockInstruction) Name() string           { return ins.name }
func (ins *mockInstruction) OpcodeID() ast.OpcodeID { return ast.OpcodeID{} }
func (ins *mockInstruction) Opcodes() []byte        { return ins.opcodes }
func (ins *mockInstruction) Size() int              { return ins.size }
func (ins *mockInstruction) SetAddress(address uint64) {
	ins.address = address
}
func (ins *mockInstruction) SetAddressing(addressing int) {
	ins.addressing = addressing
}
func (ins *mockInstruction) SetOpcodes(opcodes []byte) {
	ins.opcodes = opcodes
}
func (ins *mockInstruction) SetSize(size int) {
	ins.size = size
}
