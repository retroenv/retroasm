package assembler

import (
	"github.com/retroenv/retroasm/pkg/arch"
	m68000parser "github.com/retroenv/retroasm/pkg/arch/m68000/parser"
	"github.com/retroenv/retrogolib/arch/cpu/m68000"
)

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
