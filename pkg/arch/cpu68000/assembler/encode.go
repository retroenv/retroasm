package assembler

import (
	"encoding/binary"
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/cpu68000/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu68000"
)

// encodeEAField returns the 6-bit EA field (mode:3, reg:3) for the opcode word.
func encodeEAField(ea *parser.EffectiveAddress) (mode, reg uint8) {
	switch ea.Mode {
	case cpu68000.DataRegDirectMode:
		return 0, ea.Register
	case cpu68000.AddrRegDirectMode:
		return 1, ea.Register
	case cpu68000.AddrRegIndirectMode:
		return 2, ea.Register
	case cpu68000.PostIncrementMode:
		return 3, ea.Register
	case cpu68000.PreDecrementMode:
		return 4, ea.Register
	case cpu68000.DisplacementMode:
		return 5, ea.Register
	case cpu68000.IndexedMode:
		return 6, ea.Register
	case cpu68000.AbsShortMode:
		return 7, 0
	case cpu68000.AbsLongMode:
		return 7, 1
	case cpu68000.PCDisplacementMode:
		return 7, 2
	case cpu68000.PCIndexedMode:
		return 7, 3
	case cpu68000.ImmediateMode:
		return 7, 4
	default:
		return 0, 0
	}
}

// appendEAExtensionWords appends the extension words for an EA to the opcode buffer.
func appendEAExtensionWords(
	buf []byte,
	assigner arch.AddressAssigner,
	ea *parser.EffectiveAddress,
	opSize cpu68000.OperandSize,
) ([]byte, error) {

	if ea == nil {
		return buf, nil
	}

	switch ea.Mode {
	case cpu68000.DataRegDirectMode, cpu68000.AddrRegDirectMode,
		cpu68000.AddrRegIndirectMode, cpu68000.PostIncrementMode,
		cpu68000.PreDecrementMode, cpu68000.StatusRegMode,
		cpu68000.QuickImmediateMode:
		return buf, nil

	case cpu68000.DisplacementMode, cpu68000.PCDisplacementMode:
		return appendDisplacementEAExtension(buf, assigner, ea, opSize)

	case cpu68000.IndexedMode, cpu68000.PCIndexedMode:
		return appendIndexedEAExtension(buf, assigner, ea, opSize)

	case cpu68000.AbsShortMode, cpu68000.AbsLongMode:
		return appendAbsoluteEAExtension(buf, assigner, ea, opSize)

	case cpu68000.ImmediateMode:
		return appendImmediateEAExtension(buf, assigner, ea, opSize)

	default:
		return buf, nil
	}
}

func appendDisplacementEAExtension(
	buf []byte,
	assigner arch.AddressAssigner,
	ea *parser.EffectiveAddress,
	opSize cpu68000.OperandSize,
) ([]byte, error) {

	v, err := eaValue(assigner, ea)
	if err != nil {
		return nil, err
	}
	byteOffset := uint64(len(buf))
	if ea.Mode == cpu68000.PCDisplacementMode && effectiveAddressReferencesSymbol(ea) {
		v, err = pcRelativeValue(assigner, v, byteOffset, 16)
		if err != nil {
			return nil, err
		}
	}

	buf = binary.BigEndian.AppendUint16(buf, uint16(v))
	recordEAExtensionRelocation(assigner, ea, opSize, byteOffset)
	return buf, nil
}

func appendIndexedEAExtension(
	buf []byte,
	assigner arch.AddressAssigner,
	ea *parser.EffectiveAddress,
	opSize cpu68000.OperandSize,
) ([]byte, error) {

	byteOffset := uint64(len(buf))
	buf, err := appendIndexExtensionWord(buf, assigner, ea)
	if err != nil {
		return nil, err
	}
	recordEAExtensionRelocation(assigner, ea, opSize, byteOffset)
	return buf, nil
}

func appendAbsoluteEAExtension(
	buf []byte,
	assigner arch.AddressAssigner,
	ea *parser.EffectiveAddress,
	opSize cpu68000.OperandSize,
) ([]byte, error) {

	v, err := eaValue(assigner, ea)
	if err != nil {
		return nil, err
	}
	byteOffset := uint64(len(buf))
	if ea.Mode == cpu68000.AbsLongMode {
		buf = binary.BigEndian.AppendUint32(buf, uint32(v))
	} else {
		buf = binary.BigEndian.AppendUint16(buf, uint16(v))
	}
	recordEAExtensionRelocation(assigner, ea, opSize, byteOffset)
	return buf, nil
}

func appendImmediateEAExtension(
	buf []byte,
	assigner arch.AddressAssigner,
	ea *parser.EffectiveAddress,
	opSize cpu68000.OperandSize,
) ([]byte, error) {

	byteOffset := uint64(len(buf))
	buf, err := appendImmediateExtension(buf, assigner, ea, opSize)
	if err != nil {
		return nil, err
	}
	recordEAExtensionRelocation(assigner, ea, opSize, byteOffset)
	return buf, nil
}

func appendIndexExtensionWord(buf []byte, assigner arch.AddressAssigner, ea *parser.EffectiveAddress) ([]byte, error) {
	disp, err := eaValue(assigner, ea)
	if err != nil {
		return nil, err
	}
	if ea.Mode == cpu68000.PCIndexedMode && effectiveAddressReferencesSymbol(ea) {
		disp, err = pcRelativeValue(assigner, disp, uint64(len(buf)), 8)
		if err != nil {
			return nil, err
		}
	}

	// Brief extension word format:
	// Bit 15: D/A (0=Dn, 1=An)
	// Bits 14-12: index register number
	// Bit 11: W/L (0=.W, 1=.L)
	// Bits 10-8: 000 (reserved)
	// Bits 7-0: signed 8-bit displacement
	var ext uint16
	if ea.IsAddrReg {
		ext |= 1 << 15
	}
	ext |= uint16(ea.IndexReg&7) << 12
	if ea.IndexSize == cpu68000.SizeLong {
		ext |= 1 << 11
	}
	ext |= uint16(disp) & 0xFF

	return binary.BigEndian.AppendUint16(buf, ext), nil
}

func recordEAExtensionRelocation(
	assigner arch.AddressAssigner,
	ea *parser.EffectiveAddress,
	opSize cpu68000.OperandSize,
	byteOffset uint64,
) {

	if ea == nil || ea.Value == nil {
		return
	}

	kind := ast.AbsoluteRelocation
	width := ast.WidthWord
	switch ea.Mode {
	case cpu68000.IndexedMode:
		byteOffset++
		width = ast.WidthByte
	case cpu68000.PCIndexedMode:
		byteOffset++
		kind = ast.RelativeRelocation
		width = ast.WidthByte
	case cpu68000.PCDisplacementMode:
		kind = ast.RelativeRelocation
	case cpu68000.AbsLongMode:
		width = ast.WidthLong
	case cpu68000.ImmediateMode:
		switch opSize {
		case cpu68000.SizeByte:
			byteOffset++
			width = ast.WidthByte
		case cpu68000.SizeLong:
			width = ast.WidthLong
		}
	}
	recordCPU68000Relocation(assigner, ea.Value, cpu68000RelocationEncoding(byteOffset, kind, width))
}

func effectiveAddressReferencesSymbol(ea *parser.EffectiveAddress) bool {
	if ea == nil || ea.Value == nil {
		return false
	}
	if ast.SymbolName(ea.Value) != "" {
		return true
	}
	expression, ok := ea.Value.(ast.Expression)
	if !ok {
		return false
	}
	_, _, ok = ast.ParseSymbolReference(expression.Value)
	return ok
}

func pcRelativeValue(assigner arch.AddressAssigner, target, byteOffset uint64, bits int) (uint64, error) {
	context, ok := assigner.(*instructionRelocationAssigner)
	if !ok {
		return target, nil
	}

	displacement := int64(target) - int64(context.instruction.Address()+byteOffset)
	minimum := -(int64(1) << (bits - 1))
	maximum := (int64(1) << (bits - 1)) - 1
	if displacement < minimum || displacement > maximum {
		return 0, fmt.Errorf("PC-relative displacement %d exceeds %d-bit field", displacement, bits)
	}
	if displacement < 0 {
		return uint64((int64(1) << bits) + displacement), nil
	}
	return uint64(displacement), nil
}

func appendImmediateExtension(
	buf []byte,
	assigner arch.AddressAssigner,
	ea *parser.EffectiveAddress,
	opSize cpu68000.OperandSize,
) ([]byte, error) {

	v, err := eaValue(assigner, ea)
	if err != nil {
		return nil, err
	}

	if opSize == cpu68000.SizeLong {
		return binary.BigEndian.AppendUint32(buf, uint32(v)), nil
	}
	// Byte and word both use a word extension
	return binary.BigEndian.AppendUint16(buf, uint16(v)), nil
}

func eaValue(assigner arch.AddressAssigner, ea *parser.EffectiveAddress) (uint64, error) {
	if ea.Value == nil {
		return 0, nil
	}
	v, err := assigner.ArgumentValue(ea.Value)
	if err != nil {
		return 0, fmt.Errorf("resolving EA value: %w", err)
	}
	if ea.Negative {
		v = uint64(-int64(v))
	}
	return v, nil
}

// encodeSizeBits returns the 2-bit size field encoding for standard instructions.
func encodeSizeBits(size cpu68000.OperandSize) uint16 {
	switch size {
	case cpu68000.SizeByte:
		return 0
	case cpu68000.SizeWord:
		return 1
	case cpu68000.SizeLong:
		return 2
	default:
		return 1 // default word
	}
}

// reverseRegisterList reverses the bit order of a 16-bit register list mask.
// Used for MOVEM with pre-decrement mode where the register list is stored in reverse.
func reverseRegisterList(mask uint16) uint16 {
	var result uint16
	for i := range 16 {
		if mask&(1<<uint(i)) != 0 {
			result |= 1 << uint(15-i)
		}
	}
	return result
}
