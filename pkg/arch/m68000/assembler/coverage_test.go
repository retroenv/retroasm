package assembler

import (
	"fmt"
	"slices"
	"testing"

	m68000parser "github.com/retroenv/retroasm/pkg/arch/m68000/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/m68000"
	"github.com/retroenv/retrogolib/assert"
)

const coveragePC = uint64(0x1000)
const coverageLabelAddr = uint64(0x1010) // used for branch/DBcc targets

// TestInstructionCoverage_AllMnemonics verifies that every M68000 instruction name
// can be assembled without error and that the encoded byte count matches the computed size.
func TestInstructionCoverage_AllMnemonics(t *testing.T) {
	names := sortedInstructionNames()
	assert.NotEmpty(t, names)

	for index, name := range names {
		ins := m68000.Instructions[name]
		resolved, err := coverageResolvedInstruction(ins)
		assert.NoError(t, err, "[%03d] %s: cannot construct resolved instruction", index, name)

		t.Run(fmt.Sprintf("%03d_%s", index, name), func(t *testing.T) {
			assigner := &mockAssigner{
				pc:     coveragePC,
				values: map[string]uint64{"loop": coverageLabelAddr},
			}
			mockIns := &mockInstruction{
				name:     name,
				address:  coveragePC,
				argument: resolved,
			}

			nextPC, err := AssignInstructionAddress(assigner, mockIns)
			assert.NoError(t, err, "AssignInstructionAddress: instruction=%s", name)
			assert.True(t, mockIns.Size() >= 2, "size must be ≥ 2 bytes, got %d", mockIns.Size())
			assert.Equal(t, coveragePC+uint64(mockIns.Size()), nextPC)

			err = GenerateInstructionOpcode(assigner, mockIns)
			assert.NoError(t, err, "GenerateInstructionOpcode: instruction=%s", name)
			assert.Len(t, mockIns.Opcodes(), mockIns.Size(),
				"opcodes length must equal size for instruction=%s", name)
		})
	}
}

func sortedInstructionNames() []string {
	names := make([]string, 0, len(m68000.Instructions))
	for name := range m68000.Instructions {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// coverageResolvedInstruction constructs a minimal valid ResolvedInstruction
// for each M68000 instruction name, using register D0/A0 and small constants.
func coverageResolvedInstruction(ins *m68000.Instruction) (m68000parser.ResolvedInstruction, error) { //nolint:cyclop,gocyclo,funlen,maintidx // instruction coverage table requires many cases
	name := ins.Name

	dataD0 := &m68000parser.EffectiveAddress{Mode: m68000.DataRegDirectMode, Register: 0}
	dataD1 := &m68000parser.EffectiveAddress{Mode: m68000.DataRegDirectMode, Register: 1}
	addrA0 := &m68000parser.EffectiveAddress{Mode: m68000.AddrRegDirectMode, Register: 0}
	indA0 := &m68000parser.EffectiveAddress{Mode: m68000.AddrRegIndirectMode, Register: 0}
	indA1 := &m68000parser.EffectiveAddress{Mode: m68000.AddrRegIndirectMode, Register: 1}
	postA0 := &m68000parser.EffectiveAddress{Mode: m68000.PostIncrementMode, Register: 0}
	postA1 := &m68000parser.EffectiveAddress{Mode: m68000.PostIncrementMode, Register: 1}
	immOne := &m68000parser.EffectiveAddress{Mode: m68000.ImmediateMode, Value: ast.NewNumber(1)}
	immZero := &m68000parser.EffectiveAddress{Mode: m68000.ImmediateMode, Value: ast.NewNumber(0)}
	absLong := &m68000parser.EffectiveAddress{Mode: m68000.AbsLongMode, Value: ast.NewNumber(0x1000)}
	pcDisp := &m68000parser.EffectiveAddress{Mode: m68000.PCDisplacementMode, Value: ast.NewLabel("loop")}
	dispA0 := &m68000parser.EffectiveAddress{Mode: m68000.DisplacementMode, Register: 0, Value: ast.NewNumber(0)}

	r := func(src, dst *m68000parser.EffectiveAddress, sz m68000.OperandSize, extra uint16) m68000parser.ResolvedInstruction {
		return m68000parser.ResolvedInstruction{Instruction: ins, SrcEA: src, DstEA: dst, Size: sz, Extra: extra}
	}
	w := m68000.SizeWord
	l := m68000.SizeLong

	switch name {
	// No-operand instructions
	case m68000.NOPName, m68000.RTSName, m68000.RTEName, m68000.RTRName,
		m68000.RESETName, m68000.TRAPVName, m68000.ILLEGALName:
		return r(nil, nil, 0, 0), nil

	// MOVE / MOVEA
	case m68000.MOVEName:
		return r(dataD0, dataD1, w, 0), nil
	case m68000.MOVEAName:
		return r(dataD0, addrA0, w, 0), nil

	// ADD / ADDA / ADDX
	case m68000.ADDName:
		return r(dataD0, dataD1, w, 0), nil
	case m68000.ADDAName:
		return r(dataD0, addrA0, w, 0), nil
	case m68000.ADDXName:
		return r(dataD0, dataD1, w, 0), nil

	// SUB / SUBA / SUBX
	case m68000.SUBName:
		return r(dataD0, dataD1, w, 0), nil
	case m68000.SUBAName:
		return r(dataD0, addrA0, w, 0), nil
	case m68000.SUBXName:
		return r(dataD0, dataD1, w, 0), nil

	// AND / OR / EOR
	case m68000.ANDName:
		return r(dataD0, dataD1, w, 0), nil
	case m68000.ORName:
		return r(dataD0, dataD1, w, 0), nil
	case m68000.EORName:
		return r(dataD0, dataD1, w, 0), nil

	// CMP / CMPA / CMPM
	case m68000.CMPName:
		return r(dataD0, dataD1, w, 0), nil
	case m68000.CMPAName:
		return r(dataD0, addrA0, w, 0), nil
	case m68000.CMPMName:
		return r(postA0, postA1, w, 0), nil

	// Immediate ALU
	case m68000.ADDIName:
		return r(immOne, dataD0, w, 0), nil
	case m68000.SUBIName:
		return r(immOne, dataD0, w, 0), nil
	case m68000.ANDIName:
		return r(immOne, dataD0, w, 0), nil
	case m68000.ORIName:
		return r(immOne, dataD0, w, 0), nil
	case m68000.EORIName:
		return r(immOne, dataD0, w, 0), nil
	case m68000.CMPIName:
		return r(immOne, dataD0, w, 0), nil

	// Quick
	case m68000.ADDQName:
		return r(immOne, dataD0, w, 0), nil
	case m68000.SUBQName:
		return r(immOne, dataD0, w, 0), nil

	// Unary
	case m68000.CLRName, m68000.NEGName, m68000.NEGXName, m68000.NOTName, m68000.TSTName:
		return r(nil, dataD0, w, 0), nil
	case m68000.NBCDName, m68000.TASName:
		return r(nil, dataD0, 0, 0), nil

	// Branch
	case m68000.BRAName:
		return r(nil, pcDisp, w, 0), nil
	case m68000.BSRName:
		return r(nil, pcDisp, w, 0), nil
	case m68000.BccName:
		return r(nil, pcDisp, w, 7 /*EQ*/), nil

	// DBcc
	case m68000.DBccName:
		return r(dataD0, pcDisp, w, 6 /*NE*/), nil

	// Scc
	case m68000.SccName:
		return r(nil, dataD0, 0, 7 /*EQ*/), nil

	// MOVEQ
	case m68000.MOVEQName:
		return r(immZero, dataD0, l, 0), nil

	// LEA
	case m68000.LEAName:
		return r(indA1, addrA0, l, 0), nil

	// PEA
	case m68000.PEAName:
		return r(nil, indA0, 0, 0), nil

	// JMP / JSR — require a control EA (AbsLong)
	case m68000.JMPName, m68000.JSRName:
		return r(nil, absLong, 0, 0), nil

	// CHK <ea>,Dn — SrcEA=EA, DstEA.Register=Dn
	case m68000.CHKName:
		return r(dataD0, dataD1, w, 0), nil

	// SWAP / EXT
	case m68000.SWAPName:
		return r(nil, dataD0, 0, 0), nil
	case m68000.EXTName:
		return r(nil, dataD0, w, 0), nil

	// EXG
	case m68000.EXGName:
		return r(dataD0, dataD1, 0, 0), nil

	// LINK / UNLK
	case m68000.LINKName:
		return r(addrA0, &m68000parser.EffectiveAddress{Mode: m68000.ImmediateMode, Value: ast.NewNumber(0xFFFC)}, 0, 0), nil
	case m68000.UNLKName:
		return r(nil, addrA0, 0, 0), nil

	// TRAP / STOP
	case m68000.TRAPName:
		return r(immZero, nil, 0, 0), nil
	case m68000.STOPName:
		return r(&m68000parser.EffectiveAddress{Mode: m68000.ImmediateMode, Value: ast.NewNumber(0x2700)}, nil, 0, 0), nil

	// MOVEM reglist → (A0) — Extra=0 means register-to-memory
	case m68000.MOVEMName:
		return m68000parser.ResolvedInstruction{
			Instruction: ins,
			Size:        w,
			Extra:       0,
			SrcEA:       &m68000parser.EffectiveAddress{RegList: 0x00FF},
			DstEA:       indA0,
		}, nil

	// MOVEP Dn,d16(An)
	case m68000.MOVEPName:
		return r(dataD0, dispA0, w, 0), nil

	// Bit operations
	case m68000.BTSTName, m68000.BCHGName, m68000.BCLRName, m68000.BSETName:
		return r(dataD0, dataD1, 0, 0), nil

	// Mul / Div — DstEA.Register = Dn, SrcEA = operand
	case m68000.DIVUName, m68000.DIVSName, m68000.MULUName, m68000.MULSName:
		return r(dataD0, dataD1, w, 0), nil

	// BCD
	case m68000.ABCDName, m68000.SBCDName:
		return r(dataD0, dataD1, 0, 0), nil

	// Shifts / rotates — register shift with immediate count
	case m68000.ASLName, m68000.ASRName, m68000.LSLName, m68000.LSRName,
		m68000.ROLName, m68000.RORName, m68000.ROXLName, m68000.ROXRName:
		return r(immOne, dataD0, w, 0), nil

	default:
		return m68000parser.ResolvedInstruction{}, fmt.Errorf("no coverage mapping for instruction %q", name)
	}
}
