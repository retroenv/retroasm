package assembler

import (
	"fmt"
	"slices"
	"testing"

	"github.com/retroenv/retroasm/pkg/arch/cpu68000/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu68000"
	"github.com/retroenv/retrogolib/assert"
)

const coveragePC = uint64(0x1000)
const coverageLabelAddr = uint64(0x1010) // used for branch/DBcc targets

// TestInstructionCoverage_AllMnemonics verifies that every CPU68000 instruction name
// can be assembled without error and that the encoded byte count matches the computed size.
func TestInstructionCoverage_AllMnemonics(t *testing.T) {
	names := sortedInstructionNames()
	assert.NotEmpty(t, names)

	for index, name := range names {
		ins := cpu68000.Instructions[name]
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
	names := make([]string, 0, len(cpu68000.Instructions))
	for name := range cpu68000.Instructions {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// coverageResolvedInstruction constructs a minimal valid ResolvedInstruction
// for each CPU68000 instruction name, using register D0/A0 and small constants.
func coverageResolvedInstruction(ins *cpu68000.Instruction) (parser.ResolvedInstruction, error) { //nolint:cyclop,gocyclo,funlen,maintidx // instruction coverage table requires many cases
	name := ins.Name

	dataD0 := &parser.EffectiveAddress{Mode: cpu68000.DataRegDirectMode, Register: 0}
	dataD1 := &parser.EffectiveAddress{Mode: cpu68000.DataRegDirectMode, Register: 1}
	addrA0 := &parser.EffectiveAddress{Mode: cpu68000.AddrRegDirectMode, Register: 0}
	indA0 := &parser.EffectiveAddress{Mode: cpu68000.AddrRegIndirectMode, Register: 0}
	indA1 := &parser.EffectiveAddress{Mode: cpu68000.AddrRegIndirectMode, Register: 1}
	postA0 := &parser.EffectiveAddress{Mode: cpu68000.PostIncrementMode, Register: 0}
	postA1 := &parser.EffectiveAddress{Mode: cpu68000.PostIncrementMode, Register: 1}
	immOne := &parser.EffectiveAddress{Mode: cpu68000.ImmediateMode, Value: ast.NewNumber(1)}
	immZero := &parser.EffectiveAddress{Mode: cpu68000.ImmediateMode, Value: ast.NewNumber(0)}
	absLong := &parser.EffectiveAddress{Mode: cpu68000.AbsLongMode, Value: ast.NewNumber(0x1000)}
	pcDisp := &parser.EffectiveAddress{Mode: cpu68000.PCDisplacementMode, Value: ast.NewLabel("loop")}
	dispA0 := &parser.EffectiveAddress{Mode: cpu68000.DisplacementMode, Register: 0, Value: ast.NewNumber(0)}

	r := func(src, dst *parser.EffectiveAddress, sz cpu68000.OperandSize, extra uint16) parser.ResolvedInstruction {
		return parser.ResolvedInstruction{Instruction: ins, SrcEA: src, DstEA: dst, Size: sz, Extra: extra}
	}
	w := cpu68000.SizeWord
	l := cpu68000.SizeLong

	switch name {
	// No-operand instructions
	case cpu68000.NOPName, cpu68000.RTSName, cpu68000.RTEName, cpu68000.RTRName,
		cpu68000.RESETName, cpu68000.TRAPVName, cpu68000.ILLEGALName:
		return r(nil, nil, 0, 0), nil

	// MOVE / MOVEA
	case cpu68000.MOVEName:
		return r(dataD0, dataD1, w, 0), nil
	case cpu68000.MOVEAName:
		return r(dataD0, addrA0, w, 0), nil

	// ADD / ADDA / ADDX
	case cpu68000.ADDName:
		return r(dataD0, dataD1, w, 0), nil
	case cpu68000.ADDAName:
		return r(dataD0, addrA0, w, 0), nil
	case cpu68000.ADDXName:
		return r(dataD0, dataD1, w, 0), nil

	// SUB / SUBA / SUBX
	case cpu68000.SUBName:
		return r(dataD0, dataD1, w, 0), nil
	case cpu68000.SUBAName:
		return r(dataD0, addrA0, w, 0), nil
	case cpu68000.SUBXName:
		return r(dataD0, dataD1, w, 0), nil

	// AND / OR / EOR
	case cpu68000.ANDName:
		return r(dataD0, dataD1, w, 0), nil
	case cpu68000.ORName:
		return r(dataD0, dataD1, w, 0), nil
	case cpu68000.EORName:
		return r(dataD0, dataD1, w, 0), nil

	// CMP / CMPA / CMPM
	case cpu68000.CMPName:
		return r(dataD0, dataD1, w, 0), nil
	case cpu68000.CMPAName:
		return r(dataD0, addrA0, w, 0), nil
	case cpu68000.CMPMName:
		return r(postA0, postA1, w, 0), nil

	// Immediate ALU
	case cpu68000.ADDIName:
		return r(immOne, dataD0, w, 0), nil
	case cpu68000.SUBIName:
		return r(immOne, dataD0, w, 0), nil
	case cpu68000.ANDIName:
		return r(immOne, dataD0, w, 0), nil
	case cpu68000.ORIName:
		return r(immOne, dataD0, w, 0), nil
	case cpu68000.EORIName:
		return r(immOne, dataD0, w, 0), nil
	case cpu68000.CMPIName:
		return r(immOne, dataD0, w, 0), nil

	// Quick
	case cpu68000.ADDQName:
		return r(immOne, dataD0, w, 0), nil
	case cpu68000.SUBQName:
		return r(immOne, dataD0, w, 0), nil

	// Unary
	case cpu68000.CLRName, cpu68000.NEGName, cpu68000.NEGXName, cpu68000.NOTName, cpu68000.TSTName:
		return r(nil, dataD0, w, 0), nil
	case cpu68000.NBCDName, cpu68000.TASName:
		return r(nil, dataD0, 0, 0), nil

	// Branch
	case cpu68000.BRAName:
		return r(nil, pcDisp, w, 0), nil
	case cpu68000.BSRName:
		return r(nil, pcDisp, w, 0), nil
	case cpu68000.BccName:
		return r(nil, pcDisp, w, 7 /*EQ*/), nil

	// DBcc
	case cpu68000.DBccName:
		return r(dataD0, pcDisp, w, 6 /*NE*/), nil

	// Scc
	case cpu68000.SccName:
		return r(nil, dataD0, 0, 7 /*EQ*/), nil

	// MOVEQ
	case cpu68000.MOVEQName:
		return r(immZero, dataD0, l, 0), nil

	// LEA
	case cpu68000.LEAName:
		return r(indA1, addrA0, l, 0), nil

	// PEA
	case cpu68000.PEAName:
		return r(nil, indA0, 0, 0), nil

	// JMP / JSR — require a control EA (AbsLong)
	case cpu68000.JMPName, cpu68000.JSRName:
		return r(nil, absLong, 0, 0), nil

	// CHK <ea>,Dn — SrcEA=EA, DstEA.Register=Dn
	case cpu68000.CHKName:
		return r(dataD0, dataD1, w, 0), nil

	// SWAP / EXT
	case cpu68000.SWAPName:
		return r(nil, dataD0, 0, 0), nil
	case cpu68000.EXTName:
		return r(nil, dataD0, w, 0), nil

	// EXG
	case cpu68000.EXGName:
		return r(dataD0, dataD1, 0, 0), nil

	// LINK / UNLK
	case cpu68000.LINKName:
		return r(addrA0, &parser.EffectiveAddress{Mode: cpu68000.ImmediateMode, Value: ast.NewNumber(0xFFFC)}, 0, 0), nil
	case cpu68000.UNLKName:
		return r(nil, addrA0, 0, 0), nil

	// TRAP / STOP
	case cpu68000.TRAPName:
		return r(immZero, nil, 0, 0), nil
	case cpu68000.STOPName:
		return r(&parser.EffectiveAddress{Mode: cpu68000.ImmediateMode, Value: ast.NewNumber(0x2700)}, nil, 0, 0), nil

	// MOVEM reglist → (A0) — Extra=0 means register-to-memory
	case cpu68000.MOVEMName:
		return parser.ResolvedInstruction{
			Instruction: ins,
			Size:        w,
			Extra:       0,
			SrcEA:       &parser.EffectiveAddress{RegList: 0x00FF},
			DstEA:       indA0,
		}, nil

	// MOVEP Dn,d16(An)
	case cpu68000.MOVEPName:
		return r(dataD0, dispA0, w, 0), nil

	// Bit operations
	case cpu68000.BTSTName, cpu68000.BCHGName, cpu68000.BCLRName, cpu68000.BSETName:
		return r(dataD0, dataD1, 0, 0), nil

	// Mul / Div — DstEA.Register = Dn, SrcEA = operand
	case cpu68000.DIVUName, cpu68000.DIVSName, cpu68000.MULUName, cpu68000.MULSName:
		return r(dataD0, dataD1, w, 0), nil

	// BCD
	case cpu68000.ABCDName, cpu68000.SBCDName:
		return r(dataD0, dataD1, 0, 0), nil

	// Shifts / rotates — register shift with immediate count
	case cpu68000.ASLName, cpu68000.ASRName, cpu68000.LSLName, cpu68000.LSRName,
		cpu68000.ROLName, cpu68000.RORName, cpu68000.ROXLName, cpu68000.ROXRName:
		return r(immOne, dataD0, w, 0), nil

	default:
		return parser.ResolvedInstruction{}, fmt.Errorf("no coverage mapping for instruction %q", name)
	}
}
