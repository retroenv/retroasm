package parser

import (
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu68000"
)

// Operands is the typed CPU68000 operand set accepted by the codec builder.
type Operands struct {
	Destination *EffectiveAddress
	Extra       uint16
	Size        cpu68000.OperandSize
	Source      *EffectiveAddress
}

// UnaryOperands constructs a one-effective-address operand set.
func UnaryOperands(size cpu68000.OperandSize, destination *EffectiveAddress) Operands {
	return Operands{Size: size, Destination: destination}
}

// BinaryOperands constructs a source/destination effective-address pair.
func BinaryOperands(
	size cpu68000.OperandSize,
	source, destination *EffectiveAddress,
) Operands {

	return Operands{Size: size, Source: source, Destination: destination}
}

// DataRegister constructs a Dn effective address.
func DataRegister(register uint8) *EffectiveAddress {
	return &EffectiveAddress{Mode: cpu68000.DataRegDirectMode, Register: register}
}

// AddressRegister constructs an An effective address.
func AddressRegister(register uint8) *EffectiveAddress {
	return &EffectiveAddress{Mode: cpu68000.AddrRegDirectMode, Register: register}
}

// Indirect constructs an (An) effective address.
func Indirect(register uint8) *EffectiveAddress {
	return &EffectiveAddress{Mode: cpu68000.AddrRegIndirectMode, Register: register}
}

// PostIncrement constructs an (An)+ effective address.
func PostIncrement(register uint8) *EffectiveAddress {
	return &EffectiveAddress{Mode: cpu68000.PostIncrementMode, Register: register}
}

// PreDecrement constructs a -(An) effective address.
func PreDecrement(register uint8) *EffectiveAddress {
	return &EffectiveAddress{Mode: cpu68000.PreDecrementMode, Register: register}
}

// Displacement constructs a d16(An) effective address.
func Displacement(register uint8, value ast.Node) *EffectiveAddress {
	return &EffectiveAddress{Mode: cpu68000.DisplacementMode, Register: register, Value: value}
}

// Indexed constructs a d8(An,Xn.size) effective address.
func Indexed(
	register, indexRegister uint8,
	indexAddressRegister bool,
	indexSize cpu68000.OperandSize,
	value ast.Node,
) *EffectiveAddress {

	return &EffectiveAddress{
		Mode:      cpu68000.IndexedMode,
		Register:  register,
		IndexReg:  indexRegister,
		IndexSize: indexSize,
		IsAddrReg: indexAddressRegister,
		Value:     value,
	}
}

// Absolute constructs a short or long absolute effective address.
func Absolute(long bool, value ast.Node) *EffectiveAddress {
	mode := cpu68000.AbsShortMode
	if long {
		mode = cpu68000.AbsLongMode
	}
	return &EffectiveAddress{Mode: mode, Value: value}
}

// PCDisplacement constructs a d16(PC) effective address.
func PCDisplacement(value ast.Node) *EffectiveAddress {
	return &EffectiveAddress{Mode: cpu68000.PCDisplacementMode, Register: regPC, Value: value}
}

// PCIndexed constructs a d8(PC,Xn.size) effective address.
func PCIndexed(
	indexRegister uint8,
	indexAddressRegister bool,
	indexSize cpu68000.OperandSize,
	value ast.Node,
) *EffectiveAddress {

	return &EffectiveAddress{
		Mode:      cpu68000.PCIndexedMode,
		Register:  regPC,
		IndexReg:  indexRegister,
		IndexSize: indexSize,
		IsAddrReg: indexAddressRegister,
		Value:     value,
	}
}

// Immediate constructs a #value effective address.
func Immediate(value ast.Node) *EffectiveAddress {
	return &EffectiveAddress{Mode: cpu68000.ImmediateMode, Value: value}
}

// RegisterList constructs a MOVEM register-list effective address.
func RegisterList(mask uint16) *EffectiveAddress {
	return &EffectiveAddress{RegList: mask}
}
