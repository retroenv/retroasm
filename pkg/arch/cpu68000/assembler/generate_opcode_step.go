package assembler

import (
	"encoding/binary"
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/cpu68000/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu68000"
)

type instructionRelocationAssigner struct {
	arch.AddressAssigner

	instruction arch.Instruction
}

// GenerateInstructionOpcode generates CPU68000 opcode bytes for an already resolved instruction.
func GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	resolved, err := resolvedInstruction(ins.Argument())
	if err != nil {
		return fmt.Errorf("resolving instruction argument: %w", err)
	}

	encodingAssigner := &instructionRelocationAssigner{AddressAssigner: assigner, instruction: ins}
	opcodes, err := encodeInstruction(encodingAssigner, ins, resolved)
	if err != nil {
		return fmt.Errorf("encoding instruction '%s': %w", ins.Name(), err)
	}

	ins.SetOpcodes(opcodes)
	ins.SetSize(len(opcodes))
	return nil
}

func (ass *instructionRelocationAssigner) recordRelocation(argument any, encoding arch.RelocationEncoding) {
	arch.RecordInstructionRelocation(ass.AddressAssigner, ass.instruction, argument, encoding)
}

func recordCPU68000Relocation(assigner arch.AddressAssigner, argument any, encoding arch.RelocationEncoding) {
	recorder, ok := assigner.(*instructionRelocationAssigner)
	if ok {
		recorder.recordRelocation(argument, encoding)
	}
}

func cpu68000RelocationEncoding(byteOffset uint64, kind ast.RelocationKind, width ast.DataWidth) arch.RelocationEncoding {
	return arch.RelocationEncoding{
		ByteOffset:    byteOffset,
		Kind:          kind,
		Width:         width,
		ByteOrder:     ast.ByteOrderBig,
		ReferenceType: ast.FullAddress,
	}
}

func encodeInstruction(assigner arch.AddressAssigner, ins arch.Instruction, resolved parser.ResolvedInstruction) ([]byte, error) { //nolint:cyclop,gocyclo,funlen,maintidx // instruction encoding requires many cases
	name := resolved.Instruction.Name

	switch name {
	case cpu68000.NOPName:
		return encodeWord(0x4E71), nil
	case cpu68000.RTSName:
		return encodeWord(0x4E75), nil
	case cpu68000.RTEName:
		return encodeWord(0x4E73), nil
	case cpu68000.RTRName:
		return encodeWord(0x4E77), nil
	case cpu68000.RESETName:
		return encodeWord(0x4E70), nil
	case cpu68000.TRAPVName:
		return encodeWord(0x4E76), nil
	case cpu68000.ILLEGALName:
		return encodeWord(0x4AFC), nil

	case cpu68000.MOVEName, cpu68000.MOVEAName:
		return encodeMOVE(assigner, resolved)

	case cpu68000.ADDName, cpu68000.ADDAName, cpu68000.ADDXName:
		return encodeAddSub(assigner, resolved, 0xD000)
	case cpu68000.SUBName, cpu68000.SUBAName, cpu68000.SUBXName:
		return encodeAddSub(assigner, resolved, 0x9000)

	case cpu68000.ANDName:
		return encodeLogical(assigner, resolved, 0xC000)
	case cpu68000.ORName:
		return encodeLogical(assigner, resolved, 0x8000)
	case cpu68000.EORName:
		return encodeEOR(assigner, resolved)

	case cpu68000.CMPName, cpu68000.CMPAName:
		return encodeCMP(assigner, resolved)
	case cpu68000.CMPMName:
		return encodeCMPM(resolved)

	case cpu68000.ADDIName:
		return encodeImmediate(assigner, resolved, 0x0600)
	case cpu68000.SUBIName:
		return encodeImmediate(assigner, resolved, 0x0400)
	case cpu68000.ANDIName:
		return encodeImmediate(assigner, resolved, 0x0200)
	case cpu68000.ORIName:
		return encodeImmediate(assigner, resolved, 0x0000)
	case cpu68000.EORIName:
		return encodeImmediate(assigner, resolved, 0x0A00)
	case cpu68000.CMPIName:
		return encodeImmediate(assigner, resolved, 0x0C00)

	case cpu68000.ADDQName:
		return encodeQuick(assigner, resolved, 0x5000)
	case cpu68000.SUBQName:
		return encodeQuick(assigner, resolved, 0x5100)

	case cpu68000.CLRName:
		return encodeUnary(assigner, resolved, 0x4200)
	case cpu68000.NEGName:
		return encodeUnary(assigner, resolved, 0x4400)
	case cpu68000.NEGXName:
		return encodeUnary(assigner, resolved, 0x4000)
	case cpu68000.NOTName:
		return encodeUnary(assigner, resolved, 0x4600)
	case cpu68000.TSTName:
		return encodeUnary(assigner, resolved, 0x4A00)
	case cpu68000.NBCDName:
		return encodeUnaryByte(assigner, resolved, 0x4800)
	case cpu68000.TASName:
		return encodeUnaryByte(assigner, resolved, 0x4AC0)

	case cpu68000.BccName, cpu68000.BRAName, cpu68000.BSRName:
		return encodeBranch(assigner, ins, resolved)
	case cpu68000.DBccName:
		return encodeDBcc(assigner, ins, resolved)
	case cpu68000.SccName:
		return encodeScc(assigner, resolved)

	case cpu68000.MOVEQName:
		return encodeMOVEQ(assigner, resolved)

	case cpu68000.LEAName:
		return encodeLEA(assigner, resolved)
	case cpu68000.PEAName:
		return encodePEA(assigner, resolved)
	case cpu68000.JMPName:
		return encodeJMPJSR(assigner, resolved, 0x4EC0)
	case cpu68000.JSRName:
		return encodeJMPJSR(assigner, resolved, 0x4E80)
	case cpu68000.CHKName:
		return encodeCHK(assigner, resolved)

	case cpu68000.SWAPName:
		return encodeSWAP(resolved)
	case cpu68000.EXTName:
		return encodeEXT(resolved)
	case cpu68000.EXGName:
		return encodeEXG(resolved)

	case cpu68000.LINKName:
		return encodeLINK(assigner, resolved)
	case cpu68000.UNLKName:
		return encodeUNLK(resolved)
	case cpu68000.TRAPName:
		return encodeTRAP(assigner, resolved)
	case cpu68000.STOPName:
		return encodeSTOP(assigner, resolved)

	case cpu68000.MOVEMName:
		return encodeMOVEM(assigner, resolved)
	case cpu68000.MOVEPName:
		return encodeMOVEP(assigner, resolved)

	case cpu68000.BTSTName:
		return encodeBitOp(assigner, resolved, 0)
	case cpu68000.BCHGName:
		return encodeBitOp(assigner, resolved, 1)
	case cpu68000.BCLRName:
		return encodeBitOp(assigner, resolved, 2)
	case cpu68000.BSETName:
		return encodeBitOp(assigner, resolved, 3)

	case cpu68000.DIVUName:
		return encodeMulDiv(assigner, resolved, 0x80C0)
	case cpu68000.DIVSName:
		return encodeMulDiv(assigner, resolved, 0x81C0)
	case cpu68000.MULUName:
		return encodeMulDiv(assigner, resolved, 0xC0C0)
	case cpu68000.MULSName:
		return encodeMulDiv(assigner, resolved, 0xC1C0)

	case cpu68000.ABCDName:
		return encodeBCD(resolved, 0xC100)
	case cpu68000.SBCDName:
		return encodeBCD(resolved, 0x8100)

	case cpu68000.ASLName, cpu68000.ASRName, cpu68000.LSLName, cpu68000.LSRName,
		cpu68000.ROLName, cpu68000.RORName, cpu68000.ROXLName, cpu68000.ROXRName:
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
