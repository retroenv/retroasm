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

func encodeInstruction(assigner arch.AddressAssigner, ins arch.Instruction, resolved m68000parser.ResolvedInstruction) ([]byte, error) { //nolint:cyclop // instruction encoding requires many cases
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

func encodeMOVE(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	var sizeBits uint16
	switch resolved.Size {
	case m68000.SizeByte:
		sizeBits = 1 // Line 1
	case m68000.SizeLong:
		sizeBits = 2 // Line 2
	default:
		sizeBits = 3 // Line 3 (word)
	}

	srcMode, srcReg := encodeEAField(resolved.SrcEA)
	dstMode, dstReg := encodeEAField(resolved.DstEA)

	// MOVEA check: destination is address register
	if resolved.Instruction.Name == m68000.MOVEAName || dstMode == 1 {
		dstMode = 1
	}

	// MOVE uses reversed dst encoding: register:mode (not mode:register)
	opcode := sizeBits<<12 | uint16(dstReg)<<9 | uint16(dstMode)<<6 | uint16(srcMode)<<3 | uint16(srcReg)

	buf := encodeWord(opcode)
	var err error
	buf, err = appendEAExtensionWords(buf, assigner, resolved.SrcEA, resolved.Size)
	if err != nil {
		return nil, err
	}
	buf, err = appendEAExtensionWords(buf, assigner, resolved.DstEA, resolved.Size)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func encodeAddSub(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction, baseOpcode uint16) ([]byte, error) {
	name := resolved.Instruction.Name

	// ADDA/SUBA
	if name == m68000.ADDAName || name == m68000.SUBAName {
		return encodeAddrRegOp(assigner, resolved, baseOpcode)
	}

	// ADDX/SUBX
	if name == m68000.ADDXName || name == m68000.SUBXName {
		return encodeExtendedOp(resolved, baseOpcode)
	}

	// ADD/SUB: determine direction
	srcMode, srcReg := encodeEAField(resolved.SrcEA)
	dstMode, dstReg := encodeEAField(resolved.DstEA)

	var opMode uint16
	var eaMode, eaReg uint8
	var dn uint16

	if dstMode == 0 {
		// <ea> op Dn -> Dn
		opMode = encodeSizeBits(resolved.Size)
		eaMode = srcMode
		eaReg = srcReg
		dn = uint16(dstReg)
	} else {
		// Dn op <ea> -> <ea>
		opMode = encodeSizeBits(resolved.Size) + 4
		eaMode = dstMode
		eaReg = dstReg
		dn = uint16(srcReg)
	}

	opcode := baseOpcode | dn<<9 | opMode<<6 | uint16(eaMode)<<3 | uint16(eaReg)
	buf := encodeWord(opcode)

	var err error
	if dstMode == 0 {
		buf, err = appendEAExtensionWords(buf, assigner, resolved.SrcEA, resolved.Size)
	} else {
		buf, err = appendEAExtensionWords(buf, assigner, resolved.DstEA, resolved.Size)
	}
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func encodeAddrRegOp(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction, baseOpcode uint16) ([]byte, error) {
	srcMode, srcReg := encodeEAField(resolved.SrcEA)
	an := uint16(resolved.DstEA.Register)

	var opMode uint16
	if resolved.Size == m68000.SizeLong {
		opMode = 7
	} else {
		opMode = 3
	}

	opcode := baseOpcode | an<<9 | opMode<<6 | uint16(srcMode)<<3 | uint16(srcReg)
	buf := encodeWord(opcode)
	buf, err := appendEAExtensionWords(buf, assigner, resolved.SrcEA, resolved.Size)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func encodeExtendedOp(resolved m68000parser.ResolvedInstruction, baseOpcode uint16) ([]byte, error) {
	rx := uint16(resolved.DstEA.Register)
	ry := uint16(resolved.SrcEA.Register)
	sizeBits := encodeSizeBits(resolved.Size)

	var rm uint16
	if resolved.SrcEA.Mode == m68000.PreDecrementMode {
		rm = 1
	}

	opcode := baseOpcode | rx<<9 | 1<<8 | sizeBits<<6 | rm<<3 | ry
	return encodeWord(opcode), nil
}

func encodeLogical(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction, baseOpcode uint16) ([]byte, error) {
	srcMode, srcReg := encodeEAField(resolved.SrcEA)
	dstMode, dstReg := encodeEAField(resolved.DstEA)

	var opMode uint16
	var eaMode, eaReg uint8
	var dn uint16

	if dstMode == 0 {
		opMode = encodeSizeBits(resolved.Size)
		eaMode = srcMode
		eaReg = srcReg
		dn = uint16(dstReg)
	} else {
		opMode = encodeSizeBits(resolved.Size) + 4
		eaMode = dstMode
		eaReg = dstReg
		dn = uint16(srcReg)
	}

	opcode := baseOpcode | dn<<9 | opMode<<6 | uint16(eaMode)<<3 | uint16(eaReg)
	buf := encodeWord(opcode)

	var err error
	if dstMode == 0 {
		buf, err = appendEAExtensionWords(buf, assigner, resolved.SrcEA, resolved.Size)
	} else {
		buf, err = appendEAExtensionWords(buf, assigner, resolved.DstEA, resolved.Size)
	}
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func encodeEOR(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	// EOR Dn,<ea>
	dn := uint16(resolved.SrcEA.Register)
	dstMode, dstReg := encodeEAField(resolved.DstEA)
	opMode := encodeSizeBits(resolved.Size) + 4

	opcode := uint16(0xB000) | dn<<9 | opMode<<6 | uint16(dstMode)<<3 | uint16(dstReg)
	buf := encodeWord(opcode)
	buf, err := appendEAExtensionWords(buf, assigner, resolved.DstEA, resolved.Size)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func encodeCMP(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	name := resolved.Instruction.Name

	if name == m68000.CMPAName {
		return encodeAddrRegOp(assigner, resolved, 0xB000)
	}

	// CMP <ea>,Dn
	srcMode, srcReg := encodeEAField(resolved.SrcEA)
	dn := uint16(resolved.DstEA.Register)
	opMode := encodeSizeBits(resolved.Size)

	opcode := uint16(0xB000) | dn<<9 | opMode<<6 | uint16(srcMode)<<3 | uint16(srcReg)
	buf := encodeWord(opcode)
	buf, err := appendEAExtensionWords(buf, assigner, resolved.SrcEA, resolved.Size)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func encodeCMPM(resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	ax := uint16(resolved.DstEA.Register)
	ay := uint16(resolved.SrcEA.Register)
	sizeBits := encodeSizeBits(resolved.Size)

	opcode := uint16(0xB108) | ax<<9 | sizeBits<<6 | ay
	return encodeWord(opcode), nil
}

func encodeImmediate(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction, baseOpcode uint16) ([]byte, error) {
	sizeBits := encodeSizeBits(resolved.Size)
	dstMode, dstReg := encodeEAField(resolved.DstEA)

	opcode := baseOpcode | sizeBits<<6 | uint16(dstMode)<<3 | uint16(dstReg)
	buf := encodeWord(opcode)

	// Append immediate value (source)
	buf, err := appendEAExtensionWords(buf, assigner, resolved.SrcEA, resolved.Size)
	if err != nil {
		return nil, err
	}
	// Append destination EA extension words
	buf, err = appendEAExtensionWords(buf, assigner, resolved.DstEA, resolved.Size)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func encodeQuick(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction, baseOpcode uint16) ([]byte, error) {
	v, err := eaValue(assigner, resolved.SrcEA)
	if err != nil {
		return nil, err
	}
	data := uint16(v) & 7
	if v == 8 {
		data = 0
	}

	sizeBits := encodeSizeBits(resolved.Size)
	dstMode, dstReg := encodeEAField(resolved.DstEA)

	opcode := baseOpcode | data<<9 | sizeBits<<6 | uint16(dstMode)<<3 | uint16(dstReg)
	buf := encodeWord(opcode)
	buf, err = appendEAExtensionWords(buf, assigner, resolved.DstEA, resolved.Size)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func encodeUnary(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction, baseOpcode uint16) ([]byte, error) {
	sizeBits := encodeSizeBits(resolved.Size)
	dstMode, dstReg := encodeEAField(resolved.DstEA)

	opcode := baseOpcode | sizeBits<<6 | uint16(dstMode)<<3 | uint16(dstReg)
	buf := encodeWord(opcode)
	buf, err := appendEAExtensionWords(buf, assigner, resolved.DstEA, resolved.Size)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func encodeUnaryByte(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction, baseOpcode uint16) ([]byte, error) {
	dstMode, dstReg := encodeEAField(resolved.DstEA)

	opcode := baseOpcode | uint16(dstMode)<<3 | uint16(dstReg)
	buf := encodeWord(opcode)
	buf, err := appendEAExtensionWords(buf, assigner, resolved.DstEA, m68000.SizeByte)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func encodeBranch(assigner arch.AddressAssigner, ins arch.Instruction, resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	cond := resolved.Extra

	if resolved.Instruction.Name == m68000.BRAName {
		cond = 0
	} else if resolved.Instruction.Name == m68000.BSRName {
		cond = 1
	}

	dest, err := eaValue(assigner, resolved.DstEA)
	if err != nil {
		return nil, err
	}

	pcAfterOpcode := ins.Address() + 2
	disp := int64(dest) - int64(pcAfterOpcode)

	if resolved.Size == m68000.SizeByte && disp >= -128 && disp <= 127 && disp != 0 {
		// Short branch: 8-bit displacement in opcode word
		opcode := uint16(0x6000) | cond<<8 | uint16(byte(disp))
		return encodeWord(opcode), nil
	}

	// Word branch: 16-bit displacement
	opcode := uint16(0x6000) | cond<<8 | 0x00 // displacement 0 means word follows
	buf := encodeWord(opcode)
	buf = binary.BigEndian.AppendUint16(buf, uint16(int16(disp)))
	return buf, nil
}

func encodeDBcc(assigner arch.AddressAssigner, ins arch.Instruction, resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	cond := resolved.Extra
	dn := uint16(resolved.SrcEA.Register)

	dest, err := eaValue(assigner, resolved.DstEA)
	if err != nil {
		return nil, err
	}

	pcAfterOpcode := ins.Address() + 2
	disp := int64(dest) - int64(pcAfterOpcode)

	opcode := uint16(0x50C8) | cond<<8 | dn
	buf := encodeWord(opcode)
	buf = binary.BigEndian.AppendUint16(buf, uint16(int16(disp)))
	return buf, nil
}

func encodeScc(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	cond := resolved.Extra
	dstMode, dstReg := encodeEAField(resolved.DstEA)

	opcode := uint16(0x50C0) | cond<<8 | uint16(dstMode)<<3 | uint16(dstReg)
	buf := encodeWord(opcode)
	buf, err := appendEAExtensionWords(buf, assigner, resolved.DstEA, m68000.SizeByte)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func encodeMOVEQ(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	dn := uint16(resolved.DstEA.Register)
	v, err := eaValue(assigner, resolved.SrcEA)
	if err != nil {
		return nil, err
	}

	opcode := uint16(0x7000) | dn<<9 | uint16(v)&0xFF
	return encodeWord(opcode), nil
}

func encodeLEA(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	an := uint16(resolved.DstEA.Register)
	srcMode, srcReg := encodeEAField(resolved.SrcEA)

	opcode := uint16(0x41C0) | an<<9 | uint16(srcMode)<<3 | uint16(srcReg)
	buf := encodeWord(opcode)
	buf, err := appendEAExtensionWords(buf, assigner, resolved.SrcEA, m68000.SizeLong)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func encodePEA(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	dstMode, dstReg := encodeEAField(resolved.DstEA)

	opcode := uint16(0x4840) | uint16(dstMode)<<3 | uint16(dstReg)
	buf := encodeWord(opcode)
	buf, err := appendEAExtensionWords(buf, assigner, resolved.DstEA, m68000.SizeLong)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func encodeJMPJSR(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction, baseOpcode uint16) ([]byte, error) {
	dstMode, dstReg := encodeEAField(resolved.DstEA)

	opcode := baseOpcode | uint16(dstMode)<<3 | uint16(dstReg)
	buf := encodeWord(opcode)
	buf, err := appendEAExtensionWords(buf, assigner, resolved.DstEA, m68000.SizeLong)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func encodeCHK(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	dn := uint16(resolved.DstEA.Register)
	srcMode, srcReg := encodeEAField(resolved.SrcEA)

	opcode := uint16(0x4180) | dn<<9 | uint16(srcMode)<<3 | uint16(srcReg)
	buf := encodeWord(opcode)
	buf, err := appendEAExtensionWords(buf, assigner, resolved.SrcEA, m68000.SizeWord)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func encodeSWAP(resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	dn := uint16(resolved.DstEA.Register)
	return encodeWord(0x4840 | dn), nil
}

func encodeEXT(resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	dn := uint16(resolved.DstEA.Register)
	if resolved.Size == m68000.SizeLong {
		return encodeWord(0x48C0 | dn), nil
	}
	return encodeWord(0x4880 | dn), nil
}

func encodeEXG(resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	rx := uint16(resolved.SrcEA.Register)
	ry := uint16(resolved.DstEA.Register)

	srcIsAddr := resolved.SrcEA.Mode == m68000.AddrRegDirectMode
	dstIsAddr := resolved.DstEA.Mode == m68000.AddrRegDirectMode

	var opMode uint16
	switch {
	case !srcIsAddr && !dstIsAddr: // Dn,Dn
		opMode = 0x40
	case srcIsAddr && dstIsAddr: // An,An
		opMode = 0x48
	default: // Dn,An
		opMode = 0x88
	}

	opcode := uint16(0xC100) | rx<<9 | opMode | ry
	return encodeWord(opcode), nil
}

func encodeLINK(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	an := uint16(resolved.SrcEA.Register)
	v, err := eaValue(assigner, resolved.DstEA)
	if err != nil {
		return nil, err
	}

	opcode := uint16(0x4E50) | an
	buf := encodeWord(opcode)
	buf = binary.BigEndian.AppendUint16(buf, uint16(int16(v)))
	return buf, nil
}

func encodeUNLK(resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	an := uint16(resolved.DstEA.Register)
	return encodeWord(0x4E58 | an), nil
}

func encodeTRAP(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	v, err := eaValue(assigner, resolved.SrcEA)
	if err != nil {
		return nil, err
	}
	return encodeWord(0x4E40 | uint16(v)&0x0F), nil
}

func encodeSTOP(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	v, err := eaValue(assigner, resolved.SrcEA)
	if err != nil {
		return nil, err
	}
	buf := encodeWord(0x4E72)
	buf = binary.BigEndian.AppendUint16(buf, uint16(v))
	return buf, nil
}

func encodeMOVEM(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	var szBit uint16
	if resolved.Size == m68000.SizeLong {
		szBit = 1
	}

	if resolved.Extra == 0 {
		// Register-to-memory: MOVEM reglist,<ea>
		dstMode, dstReg := encodeEAField(resolved.DstEA)
		opcode := uint16(0x4880) | szBit<<6 | uint16(dstMode)<<3 | uint16(dstReg)
		buf := encodeWord(opcode)

		regList := resolved.SrcEA.RegList
		if resolved.DstEA.Mode == m68000.PreDecrementMode {
			regList = reverseRegisterList(regList)
		}
		buf = binary.BigEndian.AppendUint16(buf, regList)

		buf, err := appendEAExtensionWords(buf, assigner, resolved.DstEA, resolved.Size)
		if err != nil {
			return nil, err
		}
		return buf, nil
	}

	// Memory-to-register: MOVEM <ea>,reglist
	srcMode, srcReg := encodeEAField(resolved.SrcEA)
	opcode := uint16(0x4C80) | szBit<<6 | uint16(srcMode)<<3 | uint16(srcReg)
	buf := encodeWord(opcode)
	buf = binary.BigEndian.AppendUint16(buf, resolved.DstEA.RegList)
	buf, err := appendEAExtensionWords(buf, assigner, resolved.SrcEA, resolved.Size)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func encodeMOVEP(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	var dn, an uint16
	var dir uint16

	if resolved.SrcEA.Mode == m68000.DataRegDirectMode {
		// MOVEP Dn,d16(An)
		dn = uint16(resolved.SrcEA.Register)
		an = uint16(resolved.DstEA.Register)
		dir = 1
	} else {
		// MOVEP d16(An),Dn
		an = uint16(resolved.SrcEA.Register)
		dn = uint16(resolved.DstEA.Register)
		dir = 0
	}

	var szBit uint16
	if resolved.Size == m68000.SizeLong {
		szBit = 1
	}

	opcode := uint16(0x0108) | dn<<9 | dir<<7 | szBit<<6 | an
	buf := encodeWord(opcode)

	// Append displacement
	var dispEA *m68000parser.EffectiveAddress
	if dir == 1 {
		dispEA = resolved.DstEA
	} else {
		dispEA = resolved.SrcEA
	}
	v, err := eaValue(assigner, dispEA)
	if err != nil {
		return nil, err
	}
	buf = binary.BigEndian.AppendUint16(buf, uint16(v))
	return buf, nil
}

func encodeBitOp(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction, bitOp uint16) ([]byte, error) {
	dstMode, dstReg := encodeEAField(resolved.DstEA)

	if resolved.SrcEA.Mode == m68000.DataRegDirectMode {
		// Bit operation with register: BTST Dn,<ea>
		dn := uint16(resolved.SrcEA.Register)
		opcode := uint16(0x0100) | dn<<9 | bitOp<<6 | uint16(dstMode)<<3 | uint16(dstReg)
		buf := encodeWord(opcode)
		buf, err := appendEAExtensionWords(buf, assigner, resolved.DstEA, m68000.SizeByte)
		if err != nil {
			return nil, err
		}
		return buf, nil
	}

	// Bit operation with immediate: BTST #imm,<ea>
	opcode := uint16(0x0800) | bitOp<<6 | uint16(dstMode)<<3 | uint16(dstReg)
	buf := encodeWord(opcode)
	// Append bit number as word
	v, err := eaValue(assigner, resolved.SrcEA)
	if err != nil {
		return nil, err
	}
	buf = binary.BigEndian.AppendUint16(buf, uint16(v))
	buf, err = appendEAExtensionWords(buf, assigner, resolved.DstEA, m68000.SizeByte)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func encodeMulDiv(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction, baseOpcode uint16) ([]byte, error) {
	dn := uint16(resolved.DstEA.Register)
	srcMode, srcReg := encodeEAField(resolved.SrcEA)

	opcode := baseOpcode | dn<<9 | uint16(srcMode)<<3 | uint16(srcReg)
	buf := encodeWord(opcode)
	buf, err := appendEAExtensionWords(buf, assigner, resolved.SrcEA, m68000.SizeWord)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func encodeBCD(resolved m68000parser.ResolvedInstruction, baseOpcode uint16) ([]byte, error) {
	rx := uint16(resolved.DstEA.Register)
	ry := uint16(resolved.SrcEA.Register)

	var rm uint16
	if resolved.SrcEA.Mode == m68000.PreDecrementMode {
		rm = 1
	}

	opcode := baseOpcode | rx<<9 | rm<<3 | ry
	return encodeWord(opcode), nil
}

func encodeShiftRotate(assigner arch.AddressAssigner, resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	name := resolved.Instruction.Name
	shiftType, direction := shiftTypeAndDirection(name)

	if resolved.DstEA != nil && resolved.SrcEA == nil {
		// Memory shift: size=word, count=1
		dstMode, dstReg := encodeEAField(resolved.DstEA)
		opcode := uint16(0xE0C0) | shiftType<<9 | direction<<8 | uint16(dstMode)<<3 | uint16(dstReg)
		buf := encodeWord(opcode)
		buf, err := appendEAExtensionWords(buf, assigner, resolved.DstEA, m68000.SizeWord)
		if err != nil {
			return nil, err
		}
		return buf, nil
	}

	// Register shift
	reg := uint16(resolved.DstEA.Register)
	sizeBits := encodeSizeBits(resolved.Size)

	var ir, count uint16
	if resolved.SrcEA.Mode == m68000.DataRegDirectMode {
		// Count in register
		ir = 1
		count = uint16(resolved.SrcEA.Register)
	} else {
		// Immediate count
		v, err := eaValue(assigner, resolved.SrcEA)
		if err != nil {
			return nil, err
		}
		count = uint16(v) & 7
		if v == 8 {
			count = 0
		}
	}

	opcode := uint16(0xE000) | count<<9 | direction<<8 | sizeBits<<6 | ir<<5 | shiftType<<3 | reg
	return encodeWord(opcode), nil
}

func shiftTypeAndDirection(name string) (uint16, uint16) {
	switch name {
	case m68000.ASRName:
		return 0, 0
	case m68000.ASLName:
		return 0, 1
	case m68000.LSRName:
		return 1, 0
	case m68000.LSLName:
		return 1, 1
	case m68000.ROXRName:
		return 2, 0
	case m68000.ROXLName:
		return 2, 1
	case m68000.RORName:
		return 3, 0
	case m68000.ROLName:
		return 3, 1
	default:
		return 0, 0
	}
}
