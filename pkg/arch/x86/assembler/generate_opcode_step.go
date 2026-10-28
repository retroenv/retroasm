package assembler

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	isa "github.com/retroenv/retrogolib/arch/cpu/x86"
)

// GenerateInstructionOpcode emits bytes for the previously selected instruction size.
func GenerateInstructionOpcode(assigner arch.AddressAssigner, instruction arch.Instruction) error {
	encoded, err := encode(assigner, instruction, true)
	if err != nil {
		return err
	}
	if len(encoded) != instruction.Size() {
		return fmt.Errorf("8086 instruction size changed from %d to %d", instruction.Size(), len(encoded))
	}
	instruction.SetOpcodes(encoded)

	return nil
}

// ValidateInstruction checks operands without resolving forward labels.
func ValidateInstruction(instruction ast.Instruction) error {
	var values []ast.Node
	if list, ok := instruction.Argument.(ast.InstructionArguments); ok {
		values = list.Values
	} else if instruction.Argument != nil {
		values = []ast.Node{instruction.Argument}
	}

	args := make([]any, 0, len(values))
	for _, value := range values {
		if typed, ok := value.(ast.InstructionArgument); ok {
			args = append(args, typed.Value)
		} else if number, ok := ast.NumberValue(value); ok {
			args = append(args, number)
		} else {
			args = append(args, value)
		}
	}

	_, err := encodeForm(nil, instruction.Name, args, 0, false)

	return err
}

func encode(assigner arch.AddressAssigner, instruction arch.Instruction, resolve bool) ([]byte, error) {
	return encodeForm(assigner, instruction.Name(), instruction.Argument(), instruction.Address(), resolve)
}

func encodeForm(assigner arch.AddressAssigner, mnemonic string, argument any, address uint64,
	resolve bool) ([]byte, error) {

	name := strings.ToLower(mnemonic)
	var args []any
	if list, ok := argument.([]any); ok {
		args = list
	} else if argument != nil {
		args = []any{argument}
	}

	switch name {
	case isa.CldName, isa.NopName, isa.RetName:
		if len(args) != 0 {
			break
		}
		opcodes := map[string]byte{isa.CldName: 0xfc, isa.NopName: 0x90, isa.RetName: 0xc3}
		return []byte{opcodes[name]}, nil

	case isa.IntName:
		if len(args) != 1 {
			break
		}
		value, err := immediate(assigner, args[0], 1, resolve)
		return []byte{0xcd, byte(value)}, err

	case isa.PushName, isa.PopName:
		if len(args) != 1 {
			break
		}
		register, ok := args[0].(isa.RegisterParam)
		if !ok || !register.Is16Bit() {
			break
		}
		opcode := byte(0x50)
		if name == isa.PopName {
			opcode = 0x58
		}

		return []byte{opcode + byte(register-isa.RegAX)}, nil

	case isa.CallName, isa.JmpName, isa.JzName, isa.JnzName:
		if len(args) != 1 {
			break
		}
		return relative(assigner, address, name, args[0], resolve)

	case isa.MovName, isa.AddName, isa.SubName, isa.CmpName, isa.AndName, isa.OrName, isa.XorName:
		if len(args) != 2 {
			break
		}
		return binaryInstruction(assigner, name, args, resolve)
	}

	return nil, fmt.Errorf("unsupported 8086 instruction or operands: %s", mnemonic)
}

func immediate(assigner arch.AddressAssigner, arg any, width int, resolve bool) (uint64, error) {
	if _, ok := arg.(isa.RegisterParam); ok {
		return 0, errors.New("8086 immediate cannot be a register")
	}
	if !resolve {
		return 0, nil
	}
	value, err := assigner.ArgumentValue(arg)
	if err != nil {
		return 0, fmt.Errorf("resolving 8086 operand: %w", err)
	}
	if value > (1<<uint(width*8))-1 {
		return 0, fmt.Errorf("8086 immediate %d exceeds %d bytes", value, width)
	}
	return value, nil
}

func binaryInstruction(assigner arch.AddressAssigner, name string, args []any, resolve bool) ([]byte, error) {
	destination, ok := args[0].(isa.RegisterParam)
	if !ok || !destination.Is8Bit() && !destination.Is16Bit() && !destination.IsSegment() {
		return nil, errors.New("unsupported 8086 destination register")
	}
	if source, ok := args[1].(isa.RegisterParam); ok {
		return registerInstruction(name, destination, source)
	}
	if destination.IsSegment() {
		return nil, errors.New("8086 segment register cannot receive an immediate")
	}
	width := destination.GetRegisterSize()
	value, err := immediate(assigner, args[1], width, resolve)
	if err != nil {
		return nil, err
	}
	code := byte(destination)
	if width == 2 {
		code = byte(destination - isa.RegAX)
	}

	var encoded []byte
	if name == isa.MovName {
		opcode := byte(0xb0)
		if width == 2 {
			opcode = 0xb8
		}

		encoded = []byte{opcode + code}
	} else {
		opcode := byte(0x80)
		if width == 2 {
			opcode = 0x81
		}

		extensions := map[string]byte{isa.AddName: 0, isa.OrName: 1, isa.AndName: 4, isa.SubName: 5, isa.XorName: 6, isa.CmpName: 7}
		encoded = []byte{opcode, isa.NewModRM(3, extensions[name], code).ToByte()}
	}

	if width == 1 {
		return append(encoded, byte(value)), nil
	}
	return binary.LittleEndian.AppendUint16(encoded, uint16(value)), nil
}

func registerInstruction(name string, destination, source isa.RegisterParam) ([]byte, error) {
	if destination.IsSegment() || source.IsSegment() {
		if name != isa.MovName {
			return nil, errors.New("8086 segment arithmetic is unsupported")
		}
		if destination.IsSegment() && source.Is16Bit() && destination != isa.RegCS {
			return []byte{0x8e, isa.NewModRM(3, byte(destination-isa.RegES), byte(source-isa.RegAX)).ToByte()}, nil
		}
		if source.IsSegment() && destination.Is16Bit() {
			return []byte{0x8c, isa.NewModRM(3, byte(source-isa.RegES), byte(destination-isa.RegAX)).ToByte()}, nil
		}
		return nil, errors.New("invalid 8086 segment register transfer")
	}
	if destination.GetRegisterSize() != source.GetRegisterSize() || !source.Is8Bit() && !source.Is16Bit() {
		return nil, errors.New("8086 register widths do not agree")
	}
	opcodes := map[string]byte{isa.MovName: 0x88, isa.AddName: 0, isa.OrName: 8, isa.AndName: 0x20, isa.SubName: 0x28, isa.XorName: 0x30, isa.CmpName: 0x38}
	opcode := opcodes[name]
	dst := byte(destination)
	src := byte(source)
	if destination.Is16Bit() {
		opcode++
		dst = byte(destination - isa.RegAX)
		src = byte(source - isa.RegAX)
	}

	return []byte{opcode, isa.NewModRM(3, src, dst).ToByte()}, nil
}

func relative(assigner arch.AddressAssigner, address uint64, name string, arg any, resolve bool) ([]byte, error) {
	opcode := byte(0xe8)
	size := uint64(3)
	switch name {
	case isa.JmpName:
		opcode = 0xe9

	case isa.JzName:
		opcode = 0x74
		size = 2

	case isa.JnzName:
		opcode = 0x75
		size = 2
	}

	target, err := immediate(assigner, arg, 2, resolve)
	if err != nil {
		return nil, err
	}
	displacement := int64(target) - int64(address+size)
	if size == 2 {
		if resolve && (displacement < -128 || displacement > 127) {
			return nil, fmt.Errorf("8086 conditional branch distance %d exceeds signed byte", displacement)
		}
		return []byte{opcode, byte(displacement)}, nil
	}
	return binary.LittleEndian.AppendUint16([]byte{opcode}, uint16(displacement)), nil
}
