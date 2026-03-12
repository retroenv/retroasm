package assembler

import (
	"encoding/binary"

	"github.com/retroenv/retroasm/pkg/arch"
	m68000parser "github.com/retroenv/retroasm/pkg/arch/m68000/parser"
	"github.com/retroenv/retrogolib/arch/cpu/m68000"
)

func encodeBranch(assigner arch.AddressAssigner, ins arch.Instruction, resolved m68000parser.ResolvedInstruction) ([]byte, error) {
	cond := resolved.Extra

	switch resolved.Instruction.Name {
	case m68000.BRAName:
		cond = 0
	case m68000.BSRName:
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
	opcode := uint16(0x6000) | cond<<8 // displacement 0 means word follows
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
