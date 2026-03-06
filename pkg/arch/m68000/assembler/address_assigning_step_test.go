package assembler

import (
	"fmt"
	"testing"

	"github.com/retroenv/retroasm/pkg/arch"
	m68000parser "github.com/retroenv/retroasm/pkg/arch/m68000/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/m68000"
	"github.com/retroenv/retrogolib/assert"
)

var assignAddressTests = []struct {
	name     string
	resolved m68000parser.ResolvedInstruction
	wantSize int
}{
	{
		name: "NOP is 2 bytes",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.NOPName],
		},
		wantSize: 2,
	},
	{
		name: "RTS is 2 bytes",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.RTSName],
		},
		wantSize: 2,
	},
	{
		name: "ILLEGAL is 2 bytes",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.ILLEGALName],
		},
		wantSize: 2,
	},
	{
		name: "MOVEQ is 2 bytes (immediate in opcode)",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.MOVEQName],
		},
		wantSize: 2,
	},
	{
		name: "LINK is 4 bytes (opcode + displacement)",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.LINKName],
		},
		wantSize: 4,
	},
	{
		name: "STOP is 4 bytes (opcode + immediate)",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.STOPName],
		},
		wantSize: 4,
	},
	{
		name: "MOVE.L D0,D1 is 2 bytes (register direct, no extensions)",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.MOVEName],
			Size:        m68000.SizeLong,
			SrcEA:       &m68000parser.EffectiveAddress{Mode: m68000.DataRegDirectMode, Register: 0},
			DstEA:       &m68000parser.EffectiveAddress{Mode: m68000.DataRegDirectMode, Register: 1},
		},
		wantSize: 2,
	},
	{
		name: "MOVE.L #imm,D0 is 6 bytes (opcode + long immediate)",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.MOVEName],
			Size:        m68000.SizeLong,
			SrcEA:       &m68000parser.EffectiveAddress{Mode: m68000.ImmediateMode, Value: ast.NewNumber(0)},
			DstEA:       &m68000parser.EffectiveAddress{Mode: m68000.DataRegDirectMode, Register: 0},
		},
		wantSize: 6,
	},
	{
		name: "MOVE.W #imm,D0 is 4 bytes (opcode + word immediate)",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.MOVEName],
			Size:        m68000.SizeWord,
			SrcEA:       &m68000parser.EffectiveAddress{Mode: m68000.ImmediateMode, Value: ast.NewNumber(0)},
			DstEA:       &m68000parser.EffectiveAddress{Mode: m68000.DataRegDirectMode, Register: 0},
		},
		wantSize: 4,
	},
	{
		name: "CLR.L (abs long) is 6 bytes",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.CLRName],
			Size:        m68000.SizeLong,
			DstEA:       &m68000parser.EffectiveAddress{Mode: m68000.AbsLongMode, Value: ast.NewNumber(0)},
		},
		wantSize: 6,
	},
	{
		name: "CLR.W (abs short) is 4 bytes",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.CLRName],
			Size:        m68000.SizeWord,
			DstEA:       &m68000parser.EffectiveAddress{Mode: m68000.AbsShortMode, Value: ast.NewNumber(0)},
		},
		wantSize: 4,
	},
	{
		name: "Bcc.W (word displacement branch) is 4 bytes",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.BccName],
			Size:        m68000.SizeWord,
		},
		wantSize: 4,
	},
	{
		name: "Bcc.B (byte displacement branch) is 2 bytes",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.BccName],
			Size:        m68000.SizeByte,
		},
		wantSize: 2,
	},
	{
		name: "DBcc is 4 bytes (opcode + displacement)",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.DBccName],
		},
		wantSize: 4,
	},
	{
		name: "MOVEM reg-to-mem with addr-indirect is 4 bytes (opcode + reglist)",
		resolved: m68000parser.ResolvedInstruction{
			Instruction: m68000.Instructions[m68000.MOVEMName],
			Extra:       0, // register-to-memory
			SrcEA:       &m68000parser.EffectiveAddress{RegList: 0x00FF},
			DstEA:       &m68000parser.EffectiveAddress{Mode: m68000.AddrRegIndirectMode, Register: 0},
		},
		wantSize: 4,
	},
}

func TestAssignInstructionAddress(t *testing.T) {
	for _, tt := range assignAddressTests {
		t.Run(tt.name, func(t *testing.T) {
			assigner := &mockAssigner{pc: 0x1000}
			ins := &mockInstruction{
				name:     tt.resolved.Instruction.Name,
				argument: tt.resolved,
			}

			nextPC, err := AssignInstructionAddress(assigner, ins)
			assert.NoError(t, err)
			assert.Equal(t, uint64(0x1000), ins.Address())
			assert.Equal(t, tt.wantSize, ins.Size())
			assert.Equal(t, uint64(0x1000+tt.wantSize), nextPC)
		})
	}
}

func TestAssignInstructionAddress_Errors(t *testing.T) {
	assigner := &mockAssigner{pc: 0x1000}
	ins := &mockInstruction{
		name:     "test",
		argument: "not-a-resolved-instruction",
	}

	_, err := AssignInstructionAddress(assigner, ins)
	assert.Error(t, err)
}

// mockAssigner implements arch.AddressAssigner for testing.
type mockAssigner struct {
	pc     uint64
	values map[string]uint64
}

func (m *mockAssigner) ArgumentValue(argument any) (uint64, error) {
	switch v := argument.(type) {
	case ast.Number:
		return v.Value, nil
	case ast.Label:
		if m.values != nil {
			if val, ok := m.values[v.Name]; ok {
				return val, nil
			}
		}
		return 0, fmt.Errorf("label '%s' not found", v.Name)
	case ast.Identifier:
		if m.values != nil {
			if val, ok := m.values[v.Name]; ok {
				return val, nil
			}
		}
		return 0, fmt.Errorf("identifier '%s' not found", v.Name)
	default:
		return 0, fmt.Errorf("unsupported argument type %T", argument)
	}
}

func (m *mockAssigner) ProgramCounter() uint64 { return m.pc }

func (m *mockAssigner) RelativeOffset(destination, afterInstruction uint64) (byte, error) {
	diff := int64(destination) - int64(afterInstruction)
	if diff < -128 || diff > 127 {
		return 0, fmt.Errorf("relative offset %d out of range", diff)
	}
	if diff >= 0 {
		return byte(diff), nil
	}
	return byte(256 + diff), nil
}

// mockInstruction implements arch.Instruction for testing.
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

var _ arch.AddressAssigner = (*mockAssigner)(nil)
var _ arch.Instruction = (*mockInstruction)(nil)
