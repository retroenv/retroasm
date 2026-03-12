package assembler

import (
	"encoding/binary"
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	m68000parser "github.com/retroenv/retroasm/pkg/arch/m68000/parser"
	"github.com/retroenv/retrogolib/arch/cpu/m68000"
)

// GenerateInstructionOpcode generates M68000 opcode bytes for an already resolved instruction.
func GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	resolved, err := resolvedInstruction(ins.Argument())
	if err != nil {
		return fmt.Errorf("resolving instruction argument: %w", err)
	}

	opcodes, err := encodeInstruction(assigner, ins, resolved)
	if err != nil {
		return fmt.Errorf("encoding instruction '%s': %w", ins.Name(), err)
	}

	ins.SetOpcodes(opcodes)
	ins.SetSize(len(opcodes))
	return nil
}

func encodeInstruction(assigner arch.AddressAssigner, ins arch.Instruction, resolved m68000parser.ResolvedInstruction) ([]byte, error) { //nolint:cyclop,gocyclo,funlen,maintidx // instruction encoding requires many cases
	name := resolved.Instruction.Name

	switch name {
	case m68000.NOPName:
		return encodeWord(0x4E71), nil
	case m68000.RTSName:
		return encodeWord(0x4E75), nil
	case m68000.RTEName:
		return encodeWord(0x4E73), nil
	case m68000.RTRName:
		return encodeWord(0x4E77), nil
	case m68000.RESETName:
		return encodeWord(0x4E70), nil
	case m68000.TRAPVName:
		return encodeWord(0x4E76), nil
	case m68000.ILLEGALName:
		return encodeWord(0x4AFC), nil

	case m68000.MOVEName, m68000.MOVEAName:
		return encodeMOVE(assigner, resolved)

	case m68000.ADDName, m68000.ADDAName, m68000.ADDXName:
		return encodeAddSub(assigner, resolved, 0xD000)
	case m68000.SUBName, m68000.SUBAName, m68000.SUBXName:
		return encodeAddSub(assigner, resolved, 0x9000)

	case m68000.ANDName:
		return encodeLogical(assigner, resolved, 0xC000)
	case m68000.ORName:
		return encodeLogical(assigner, resolved, 0x8000)
	case m68000.EORName:
		return encodeEOR(assigner, resolved)

	case m68000.CMPName, m68000.CMPAName:
		return encodeCMP(assigner, resolved)
	case m68000.CMPMName:
		return encodeCMPM(resolved)

	case m68000.ADDIName:
		return encodeImmediate(assigner, resolved, 0x0600)
	case m68000.SUBIName:
		return encodeImmediate(assigner, resolved, 0x0400)
	case m68000.ANDIName:
		return encodeImmediate(assigner, resolved, 0x0200)
	case m68000.ORIName:
		return encodeImmediate(assigner, resolved, 0x0000)
	case m68000.EORIName:
		return encodeImmediate(assigner, resolved, 0x0A00)
	case m68000.CMPIName:
		return encodeImmediate(assigner, resolved, 0x0C00)

	case m68000.ADDQName:
		return encodeQuick(assigner, resolved, 0x5000)
	case m68000.SUBQName:
		return encodeQuick(assigner, resolved, 0x5100)

	case m68000.CLRName:
		return encodeUnary(assigner, resolved, 0x4200)
	case m68000.NEGName:
		return encodeUnary(assigner, resolved, 0x4400)
	case m68000.NEGXName:
		return encodeUnary(assigner, resolved, 0x4000)
	case m68000.NOTName:
		return encodeUnary(assigner, resolved, 0x4600)
	case m68000.TSTName:
		return encodeUnary(assigner, resolved, 0x4A00)
	case m68000.NBCDName:
		return encodeUnaryByte(assigner, resolved, 0x4800)
	case m68000.TASName:
		return encodeUnaryByte(assigner, resolved, 0x4AC0)

	case m68000.BccName, m68000.BRAName, m68000.BSRName:
		return encodeBranch(assigner, ins, resolved)
	case m68000.DBccName:
		return encodeDBcc(assigner, ins, resolved)
	case m68000.SccName:
		return encodeScc(assigner, resolved)

	case m68000.MOVEQName:
		return encodeMOVEQ(assigner, resolved)

	case m68000.LEAName:
		return encodeLEA(assigner, resolved)
	case m68000.PEAName:
		return encodePEA(assigner, resolved)
	case m68000.JMPName:
		return encodeJMPJSR(assigner, resolved, 0x4EC0)
	case m68000.JSRName:
		return encodeJMPJSR(assigner, resolved, 0x4E80)
	case m68000.CHKName:
		return encodeCHK(assigner, resolved)

	case m68000.SWAPName:
		return encodeSWAP(resolved)
	case m68000.EXTName:
		return encodeEXT(resolved)
	case m68000.EXGName:
		return encodeEXG(resolved)

	case m68000.LINKName:
		return encodeLINK(assigner, resolved)
	case m68000.UNLKName:
		return encodeUNLK(resolved)
	case m68000.TRAPName:
		return encodeTRAP(assigner, resolved)
	case m68000.STOPName:
		return encodeSTOP(assigner, resolved)

	case m68000.MOVEMName:
		return encodeMOVEM(assigner, resolved)
	case m68000.MOVEPName:
		return encodeMOVEP(assigner, resolved)

	case m68000.BTSTName:
		return encodeBitOp(assigner, resolved, 0)
	case m68000.BCHGName:
		return encodeBitOp(assigner, resolved, 1)
	case m68000.BCLRName:
		return encodeBitOp(assigner, resolved, 2)
	case m68000.BSETName:
		return encodeBitOp(assigner, resolved, 3)

	case m68000.DIVUName:
		return encodeMulDiv(assigner, resolved, 0x80C0)
	case m68000.DIVSName:
		return encodeMulDiv(assigner, resolved, 0x81C0)
	case m68000.MULUName:
		return encodeMulDiv(assigner, resolved, 0xC0C0)
	case m68000.MULSName:
		return encodeMulDiv(assigner, resolved, 0xC1C0)

	case m68000.ABCDName:
		return encodeBCD(resolved, 0xC100)
	case m68000.SBCDName:
		return encodeBCD(resolved, 0x8100)

	case m68000.ASLName, m68000.ASRName, m68000.LSLName, m68000.LSRName,
		m68000.ROLName, m68000.RORName, m68000.ROXLName, m68000.ROXRName:
		return encodeShiftRotate(assigner, resolved)

	default:
		return nil, fmt.Errorf("unsupported instruction '%s'", name)
	}
}

func encodeWord(w uint16) []byte {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, w)
	return buf
}
