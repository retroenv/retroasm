package assembler

import (
	"encoding/binary"
	"errors"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/x86/parser"
	isa "github.com/retroenv/retrogolib/arch/cpu/x86"
)

// RelaxInstruction grows an out-of-range conditional branch to an inverse
// short branch around a near jump. A grown branch never becomes short again.
func RelaxInstruction(assigner arch.AddressAssigner, instruction arch.Instruction) (bool, error) {
	if conditionalOpcode(instruction.Name()) == 0 || instruction.Size() != 2 {
		return false, nil
	}
	target, err := immediate(assigner, instruction.Argument(), 2, true)
	if err != nil {
		return false, err
	}
	distance := int64(target) - int64(instruction.Address()+2)
	if distance >= -128 && distance <= 127 {
		return false, nil
	}
	instruction.SetSize(5)

	return true, nil
}

func conditionalOpcode(name string) byte {
	name = strings.ToLower(name)
	if !isa.ConditionalJumpInstructions.Contains(name) {
		return 0
	}
	info, known := isa.Instructions[name].GetOpcodeInfo(isa.RelativeAddressing)
	if !known || info.Opcode < 0x70 || info.Opcode > 0x7f {
		return 0
	}
	return byte(info.Opcode)
}

func controlInstruction(assigner arch.AddressAssigner, name string, args []any, address uint64,
	resolve bool) ([]byte, error) {

	if len(args) != 1 {
		return nil, errors.New("8086 branch or call needs one target")
	}
	_, register := parser.RegisterValue(args[0])
	_, memory := args[0].(parser.MemoryOperand)
	if register || memory {
		field := byte(2)
		if name == isa.JmpName {
			field = 4
		} else if name != isa.CallName {
			return nil, errors.New("8086 conditional branch needs a direct target")
		}

		return encodeRM(assigner, 0xff, field, args[0], 2, resolve)
	}
	return relative(assigner, address, name, args[0], resolve)
}

func relative(assigner arch.AddressAssigner, address uint64, name string, arg any, resolve bool) ([]byte, error) {
	opcode := conditionalOpcode(name)
	size := uint64(2)
	if opcode == 0 {
		opcode = 0xe8
		size = 3
		if name == isa.JmpName {
			opcode = 0xe9
		}
	}

	target, err := immediate(assigner, arg, 2, resolve)
	if err != nil {
		return nil, err
	}
	distance := int64(target) - int64(address+size)
	if size == 2 {
		if resolve && (distance < -128 || distance > 127) {
			return nil, errors.New("8086 conditional branch needs address relaxation")
		}
		return []byte{opcode, byte(distance)}, nil
	}
	return binary.LittleEndian.AppendUint16([]byte{opcode}, uint16(distance)), nil
}

func longConditional(assigner arch.AddressAssigner, name string, arg any, address uint64,
	resolve bool) ([]byte, error) {

	target, err := immediate(assigner, arg, 2, resolve)
	if err != nil {
		return nil, err
	}
	distance := int64(target) - int64(address+5)

	return binary.LittleEndian.AppendUint16([]byte{conditionalOpcode(name) ^ 1, 3, 0xe9}, uint16(distance)), nil
}
