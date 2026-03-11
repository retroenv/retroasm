package assembler

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/retroenv/retroasm/pkg/arch"
)

// Instruction name constants.
const (
	movInst  = "MOV"
	addInst  = "ADD"
	subInst  = "SUB"
	cmpInst  = "CMP"
	incInst  = "INC"
	decInst  = "DEC"
	jmpInst  = "JMP"
	jeInst   = "JE"
	jneInst  = "JNE"
	callInst = "CALL"
	retInst  = "RET"
	pushInst = "PUSH"
	popInst  = "POP"
	andInst  = "AND"
	orInst   = "OR"
	xorInst  = "XOR"
	nopInst  = "NOP"
)

// GenerateInstructionOpcode generates the instruction opcode for x86 instructions.
func GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	name := ins.Name()
	addressing := ins.Addressing()

	// Get base opcode for the instruction and addressing mode
	opcode := getBaseOpcode(name, addressing)
	opcodes := []byte{opcode}

	switch addressing {
	case 0: // RegisterAddressing
		if err := generateRegisterAddressingOpcode(assigner, ins, &opcodes); err != nil {
			return fmt.Errorf("generating register opcode: %w", err)
		}

	case 1: // ImmediateAddressing
		if err := generateImmediateAddressingOpcode(assigner, ins, &opcodes); err != nil {
			return fmt.Errorf("generating immediate opcode: %w", err)
		}

	case 2: // DirectAddressing
		if err := generateDirectAddressingOpcode(assigner, ins, &opcodes); err != nil {
			return fmt.Errorf("generating direct opcode: %w", err)
		}

	default:
		return fmt.Errorf("unsupported addressing mode %d", addressing)
	}

	ins.SetOpcodes(opcodes)
	return nil
}

func getBaseOpcode(name string, addressing int) byte {
	return getInstructionOpcode(name, addressing)
}

func getInstructionOpcode(name string, addressing int) byte {
	// Handle arithmetic instructions with addressing modes
	if opcode := getArithmeticOpcode(name, addressing); opcode != 0 {
		return opcode
	}

	// Handle simple single-byte instructions
	return getSimpleOpcode(name)
}

func getArithmeticOpcode(name string, addressing int) byte {
	switch name {
	case movInst:
		return getMovOpcode(addressing)
	case addInst:
		return getAddOpcode(addressing)
	case subInst:
		return getSubOpcode(addressing)
	case cmpInst:
		return getCmpOpcode(addressing)
	case andInst:
		return getAndOpcode(addressing)
	case orInst:
		return getOrOpcode(addressing)
	case xorInst:
		return getXorOpcode(addressing)
	}
	return 0 // Not an arithmetic instruction
}

func getSimpleOpcode(name string) byte {
	switch name {
	case incInst:
		return 0x40 // INC r16
	case decInst:
		return 0x48 // DEC r16
	case jmpInst:
		return 0xE9 // JMP rel16
	case jeInst:
		return 0x74 // JE rel8
	case jneInst:
		return 0x75 // JNE rel8
	case callInst:
		return 0xE8 // CALL rel16
	case retInst:
		return 0xC3 // RET
	case pushInst:
		return 0x50 // PUSH r16
	case popInst:
		return 0x58 // POP r16
	case nopInst:
		return 0x90 // NOP
	}
	return 0x90 // Default NOP
}

func getMovOpcode(addressing int) byte {
	switch addressing {
	case 0: // RegisterAddressing
		return 0x89 // MOV r/m16, r16
	case 1: // ImmediateAddressing
		return 0xB8 // MOV r16, imm16
	case 2: // DirectAddressing
		return 0x8B // MOV r16, r/m16
	}
	return 0x89
}

func getAddOpcode(addressing int) byte {
	switch addressing {
	case 0:
		return 0x01 // ADD r/m16, r16
	case 1:
		return 0x05 // ADD AX, imm16
	}
	return 0x01
}

func getSubOpcode(addressing int) byte {
	switch addressing {
	case 0:
		return 0x29 // SUB r/m16, r16
	case 1:
		return 0x2D // SUB AX, imm16
	}
	return 0x29
}

func getCmpOpcode(addressing int) byte {
	switch addressing {
	case 0:
		return 0x39 // CMP r/m16, r16
	case 1:
		return 0x3D // CMP AX, imm16
	}
	return 0x39
}

func getAndOpcode(addressing int) byte {
	switch addressing {
	case 0:
		return 0x21 // AND r/m16, r16
	case 1:
		return 0x25 // AND AX, imm16
	}
	return 0x21
}

func getOrOpcode(addressing int) byte {
	switch addressing {
	case 0:
		return 0x09 // OR r/m16, r16
	case 1:
		return 0x0D // OR AX, imm16
	}
	return 0x09
}

func getXorOpcode(addressing int) byte {
	switch addressing {
	case 0:
		return 0x31 // XOR r/m16, r16
	case 1:
		return 0x35 // XOR AX, imm16
	}
	return 0x31
}

func generateRegisterAddressingOpcode(assigner arch.AddressAssigner, ins arch.Instruction, opcodes *[]byte) error {
	name := ins.Name()

	// Handle single register instructions (INC, DEC, PUSH, POP)
	if isSingleRegisterInstruction(name) {
		if ins.Argument() != nil {
			// Get register code from argument
			value, err := assigner.ArgumentValue(ins.Argument())
			if err != nil {
				return fmt.Errorf("getting register argument: %w", err)
			}

			// Modify opcode based on register
			(*opcodes)[0] += byte(value)
		}
		return nil
	}

	// Handle two-operand register instructions that need ModR/M byte
	if needsModRM(name, 0) {
		// For simplicity, generate a basic ModR/M byte
		// Real implementation would need proper ModR/M encoding
		modrm := byte(0xC0) // 11 000 000 - register to register
		if ins.Argument() != nil {
			value, err := assigner.ArgumentValue(ins.Argument())
			if err != nil {
				return fmt.Errorf("getting register argument: %w", err)
			}
			modrm |= byte(value) // Set r/m field
		}
		*opcodes = append(*opcodes, modrm)
	}

	return nil
}

func generateImmediateAddressingOpcode(assigner arch.AddressAssigner, ins arch.Instruction, opcodes *[]byte) error {
	value, err := assigner.ArgumentValue(ins.Argument())
	if err != nil {
		return fmt.Errorf("getting immediate argument: %w", err)
	}

	name := ins.Name()

	// Handle branch instructions (relative addressing)
	if isBranchInstruction(name) {
		return generateBranchOpcode(name, value, opcodes)
	}

	// Handle immediate data
	if value > math.MaxUint16 {
		return fmt.Errorf("immediate value %d exceeds word", value)
	}

	// Determine if we need 8-bit or 16-bit immediate
	if value <= math.MaxUint8 && canUse8BitImmediate(name) {
		*opcodes = append(*opcodes, byte(value))
	} else {
		*opcodes = binary.LittleEndian.AppendUint16(*opcodes, uint16(value))
	}

	return nil
}

func generateBranchOpcode(name string, value uint64, opcodes *[]byte) error {
	if name == jmpInst || name == callInst {
		// 16-bit relative offset
		if value > math.MaxUint16 {
			return fmt.Errorf("relative offset %d exceeds word", value)
		}
		*opcodes = binary.LittleEndian.AppendUint16(*opcodes, uint16(value))
	} else {
		// 8-bit relative offset for conditional jumps
		if value > math.MaxUint8 {
			return fmt.Errorf("relative offset %d exceeds byte", value)
		}
		*opcodes = append(*opcodes, byte(value))
	}
	return nil
}

func generateDirectAddressingOpcode(assigner arch.AddressAssigner, ins arch.Instruction, opcodes *[]byte) error {
	value, err := assigner.ArgumentValue(ins.Argument())
	if err != nil {
		return fmt.Errorf("getting direct address argument: %w", err)
	}
	if value > math.MaxUint16 {
		return fmt.Errorf("address %d exceeds word", value)
	}

	// Add ModR/M byte for direct addressing (mod=00, r/m=110)
	if needsModRM(ins.Name(), 2) {
		modrm := byte(0x06) // 00 000 110 - direct addressing
		*opcodes = append(*opcodes, modrm)
	}

	// Add 16-bit address
	*opcodes = binary.LittleEndian.AppendUint16(*opcodes, uint16(value))
	return nil
}

func needsModRM(name string, addressing int) bool {
	switch name {
	case movInst, addInst, subInst, cmpInst, andInst, orInst, xorInst:
		return addressing == 0 || addressing == 2 // Register or Direct addressing
	default:
		return false
	}
}

func isSingleRegisterInstruction(name string) bool {
	switch name {
	case incInst, decInst, pushInst, popInst:
		return true
	default:
		return false
	}
}

func isBranchInstruction(name string) bool {
	switch name {
	case jmpInst, callInst, jeInst, jneInst:
		return true
	default:
		return false
	}
}

func canUse8BitImmediate(name string) bool {
	// Most arithmetic operations can use 8-bit immediates
	switch name {
	case addInst, subInst, cmpInst, andInst, orInst, xorInst:
		return true
	default:
		return false
	}
}
