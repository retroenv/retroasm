// Package x86 provides x86 (8086/286) architecture support for the assembler.
package x86

import (
	"fmt"
)

// Register represents an x86 register.
type Register struct {
	Name string
	Size RegisterSize
	Code int // Register encoding value
}

// x86 registers.
var (
	// 8-bit registers.
	AL = Register{"AL", Byte, 0}
	CL = Register{"CL", Byte, 1}
	DL = Register{"DL", Byte, 2}
	BL = Register{"BL", Byte, 3}
	AH = Register{"AH", Byte, 4}
	CH = Register{"CH", Byte, 5}
	DH = Register{"DH", Byte, 6}
	BH = Register{"BH", Byte, 7}

	// 16-bit registers.
	AX = Register{"AX", Word, 0}
	CX = Register{"CX", Word, 1}
	DX = Register{"DX", Word, 2}
	BX = Register{"BX", Word, 3}
	SP = Register{"SP", Word, 4}
	BP = Register{"BP", Word, 5}
	SI = Register{"SI", Word, 6}
	DI = Register{"DI", Word, 7}

	// Segment registers.
	ES = Register{"ES", Word, 0}
	CS = Register{"CS", Word, 1}
	SS = Register{"SS", Word, 2}
	DS = Register{"DS", Word, 3}
)

// Registers maps register names to Register structs.
var Registers = map[string]Register{
	"AL": AL, "CL": CL, "DL": DL, "BL": BL,
	"AH": AH, "CH": CH, "DH": DH, "BH": BH,
	"AX": AX, "CX": CX, "DX": DX, "BX": BX,
	"SP": SP, "BP": BP, "SI": SI, "DI": DI,
	"ES": ES, "CS": CS, "SS": SS, "DS": DS,
}

// Instructions contains all supported x86 instructions.
var Instructions = map[string]*Instruction{
	// Data Movement
	"MOV": {
		Name: "MOV",
		Addressing: map[AddressingMode]AddressingInfo{
			RegisterAddressing:  {0x89, 2, true},  // MOV r/m16, r16
			ImmediateAddressing: {0xB8, 3, false}, // MOV r16, imm16
			DirectAddressing:    {0x8B, 4, true},  // MOV r16, r/m16
		},
	},

	// Arithmetic
	"ADD": {
		Name: "ADD",
		Addressing: map[AddressingMode]AddressingInfo{
			RegisterAddressing:  {0x01, 2, true},  // ADD r/m16, r16
			ImmediateAddressing: {0x05, 3, false}, // ADD AX, imm16
		},
	},

	"SUB": {
		Name: "SUB",
		Addressing: map[AddressingMode]AddressingInfo{
			RegisterAddressing:  {0x29, 2, true},  // SUB r/m16, r16
			ImmediateAddressing: {0x2D, 3, false}, // SUB AX, imm16
		},
	},

	"CMP": {
		Name: "CMP",
		Addressing: map[AddressingMode]AddressingInfo{
			RegisterAddressing:  {0x39, 2, true},  // CMP r/m16, r16
			ImmediateAddressing: {0x3D, 3, false}, // CMP AX, imm16
		},
	},

	"INC": {
		Name: "INC",
		Addressing: map[AddressingMode]AddressingInfo{
			RegisterAddressing: {0x40, 1, false}, // INC r16
		},
	},

	"DEC": {
		Name: "DEC",
		Addressing: map[AddressingMode]AddressingInfo{
			RegisterAddressing: {0x48, 1, false}, // DEC r16
		},
	},

	// Control Flow
	"JMP": {
		Name: "JMP",
		Addressing: map[AddressingMode]AddressingInfo{
			ImmediateAddressing: {0xE9, 3, false}, // JMP rel16
		},
	},

	"JE": {
		Name: "JE",
		Addressing: map[AddressingMode]AddressingInfo{
			ImmediateAddressing: {0x74, 2, false}, // JE rel8
		},
	},

	"JNE": {
		Name: "JNE",
		Addressing: map[AddressingMode]AddressingInfo{
			ImmediateAddressing: {0x75, 2, false}, // JNE rel8
		},
	},

	"CALL": {
		Name: "CALL",
		Addressing: map[AddressingMode]AddressingInfo{
			ImmediateAddressing: {0xE8, 3, false}, // CALL rel16
		},
	},

	"RET": {
		Name: "RET",
		Addressing: map[AddressingMode]AddressingInfo{
			RegisterAddressing: {0xC3, 1, false}, // RET (implied)
		},
	},

	// Stack Operations
	"PUSH": {
		Name: "PUSH",
		Addressing: map[AddressingMode]AddressingInfo{
			RegisterAddressing: {0x50, 1, false}, // PUSH r16
		},
	},

	"POP": {
		Name: "POP",
		Addressing: map[AddressingMode]AddressingInfo{
			RegisterAddressing: {0x58, 1, false}, // POP r16
		},
	},

	// Logic Operations
	"AND": {
		Name: "AND",
		Addressing: map[AddressingMode]AddressingInfo{
			RegisterAddressing:  {0x21, 2, true},  // AND r/m16, r16
			ImmediateAddressing: {0x25, 3, false}, // AND AX, imm16
		},
	},

	"OR": {
		Name: "OR",
		Addressing: map[AddressingMode]AddressingInfo{
			RegisterAddressing:  {0x09, 2, true},  // OR r/m16, r16
			ImmediateAddressing: {0x0D, 3, false}, // OR AX, imm16
		},
	},

	"XOR": {
		Name: "XOR",
		Addressing: map[AddressingMode]AddressingInfo{
			RegisterAddressing:  {0x31, 2, true},  // XOR r/m16, r16
			ImmediateAddressing: {0x35, 3, false}, // XOR AX, imm16
		},
	},

	// Misc
	"NOP": {
		Name: "NOP",
		Addressing: map[AddressingMode]AddressingInfo{
			RegisterAddressing: {0x90, 1, false}, // NOP (implied)
		},
	},
}

// String returns a string representation of the addressing mode.
func (a AddressingMode) String() string {
	switch a {
	case RegisterAddressing:
		return "register"
	case ImmediateAddressing:
		return "immediate"
	case DirectAddressing:
		return "direct"
	case RegisterIndirect:
		return "register_indirect"
	case BasedAddressing:
		return "based"
	case IndexedAddressing:
		return "indexed"
	case BasedIndexedAddressing:
		return "based_indexed"
	default:
		return fmt.Sprintf("unknown(%d)", int(a))
	}
}
