package assembler

import (
	"encoding/binary"
	"errors"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/x86/parser"
	isa "github.com/retroenv/retrogolib/arch/cpu/x86"
)

var binaryOpcodes = map[string]byte{
	isa.AddName: 0, isa.OrName: 8, isa.AdcName: 0x10, isa.SbbName: 0x18,
	isa.AndName: 0x20, isa.SubName: 0x28, isa.XorName: 0x30, isa.CmpName: 0x38,
	isa.MovName: 0x88, isa.TestName: 0x84, isa.XchgName: 0x86,
}

func binaryInstruction(assigner arch.AddressAssigner, name string, args []any, resolve bool) ([]byte, error) {
	destination, destinationRegister := parser.RegisterValue(args[0])
	source, sourceRegister := parser.RegisterValue(args[1])
	if destinationRegister && destination.IsSegment() || sourceRegister && source.IsSegment() {
		return segmentTransfer(assigner, name, args, resolve)
	}
	if err := validateGeneralOperands(args); err != nil {
		return nil, err
	}
	if name == isa.MovName {
		encoded, handled, err := accumulatorMove(assigner, args, resolve)
		if handled {
			return encoded, err
		}
	}

	if name == isa.XchgName && destinationRegister && sourceRegister {
		return exchangeRegisters(assigner, destination, source, args[1], resolve)
	}
	if sourceRegister {
		opcode := binaryOpcodes[name]
		width := source.GetRegisterSize()
		if width == 2 {
			opcode++
		}

		return encodeRM(assigner, opcode, registerCode(source), args[0], width, resolve)
	}
	if _, memory := args[1].(parser.MemoryOperand); memory {
		if !destinationRegister {
			return nil, errors.New("8086 cannot use two memory operands")
		}
		opcode := binaryOpcodes[name]
		if name != isa.TestName && name != isa.XchgName {
			opcode += 2
		}

		width := destination.GetRegisterSize()
		if width == 2 {
			opcode++
		}

		return encodeRM(assigner, opcode, registerCode(destination), args[1], width, resolve)
	}
	return immediateInstruction(assigner, name, args[0], args[1], resolve)
}

func immediateInstruction(assigner arch.AddressAssigner, name string, destination, source any,
	resolve bool) ([]byte, error) {

	width := operandWidth(destination)
	if width != 1 && width != 2 {
		return nil, errors.New("8086 immediate destination needs an explicit width")
	}
	value, err := immediate(assigner, source, width, resolve)
	if err != nil {
		return nil, err
	}
	if register, valid := parser.RegisterValue(destination); valid &&
		(register == isa.RegAL || register == isa.RegAX) && name != isa.MovName && name != isa.XchgName {

		opcode := binaryOpcodes[name] + 4
		if name == isa.TestName {
			opcode = 0xa8
		}

		return appendImmediate([]byte{opcode + byte(width-1)}, value, width), nil
	}
	opcode := byte(0x80)
	if width == 2 {
		opcode++
	}

	field := binaryOpcodes[name] >> 3
	if name == isa.XchgName {
		return nil, errors.New("8086 XCHG cannot use an immediate")
	}
	switch name {
	case isa.MovName:
		if register, valid := parser.RegisterValue(destination); valid {
			opcode = 0xb0 + registerCode(register)
			if width == 2 {
				opcode += 8
			}

			return appendImmediate([]byte{opcode}, value, width), nil
		}
		opcode = 0xc6 + byte(width-1)
		field = 0

	case isa.TestName:
		opcode = 0xf6 + byte(width-1)
		field = 0
	}

	encoded, err := encodeRM(assigner, opcode, field, destination, width, resolve)
	if err != nil {
		return nil, err
	}
	return appendImmediate(encoded, value, width), nil
}

func appendImmediate(encoded []byte, value uint64, width int) []byte {
	if width == 1 {
		return append(encoded, byte(value))
	}
	return binary.LittleEndian.AppendUint16(encoded, uint16(value))
}

func segmentTransfer(assigner arch.AddressAssigner, name string, args []any, resolve bool) ([]byte, error) {
	if name != isa.MovName {
		return nil, errors.New("8086 segment arithmetic is unsupported")
	}
	if destination, valid := parser.RegisterValue(args[0]); valid && destination.IsSegment() {
		if destination == isa.RegCS {
			return nil, errors.New("8086 cannot load CS with MOV")
		}
		return encodeRM(assigner, 0x8e, byte(destination-isa.RegES), args[1], 2, resolve)
	}
	source, valid := parser.RegisterValue(args[1])
	if !valid || !source.IsSegment() {
		return nil, errors.New("invalid 8086 segment transfer")
	}
	return encodeRM(assigner, 0x8c, byte(source-isa.RegES), args[0], 2, resolve)
}

func accumulatorMemory(assigner arch.AddressAssigner, memory parser.MemoryOperand, width int, opcode byte,
	resolve bool) ([]byte, error) {

	if memory.Width != 0 && int(memory.Width) != width {
		return nil, errors.New("8086 memory width does not match the operation")
	}
	if _, _, _, err := memoryForm(memory); err != nil {
		return nil, err
	}
	prefix, err := segmentPrefix(memory.Segment)
	if err != nil {
		return nil, err
	}
	displacement, err := memoryDisplacement(assigner, memory, resolve)
	if err != nil {
		return nil, err
	}
	return binary.LittleEndian.AppendUint16(append(prefix, opcode+byte(width-1)), uint16(displacement)), nil
}

func accumulatorMove(assigner arch.AddressAssigner, args []any, resolve bool) ([]byte, bool, error) {
	memory, valid := args[0].(parser.MemoryOperand)
	registerValue := args[1]
	opcode := byte(0xa2)
	if !valid {
		memory, valid = args[1].(parser.MemoryOperand)
		registerValue = args[0]
		opcode = 0xa0
	}

	if !valid || memory.Address != isa.RegMem {
		return nil, false, nil
	}
	register, valid := parser.RegisterValue(registerValue)
	if !valid || register != isa.RegAL && register != isa.RegAX {
		return nil, false, nil
	}
	encoded, err := accumulatorMemory(assigner, memory, register.GetRegisterSize(), opcode, resolve)

	return encoded, true, err
}

func exchangeRegisters(assigner arch.AddressAssigner, destination, source isa.RegisterParam, operand any,
	resolve bool) ([]byte, error) {

	if destination.Is16Bit() && source.Is16Bit() && (destination == isa.RegAX || source == isa.RegAX) {
		other := destination
		if destination == isa.RegAX {
			other = source
		}

		return []byte{0x90 + registerCode(other)}, nil
	}
	width := destination.GetRegisterSize()
	return encodeRM(assigner, 0x86+byte(width-1), registerCode(destination), operand, width, resolve)
}

func validateGeneralOperands(args []any) error {
	for _, operand := range args {
		if register, valid := parser.RegisterValue(operand); valid && !register.Is8Bit() && !register.Is16Bit() {
			return errors.New("8086 binary operation needs general registers")
		}
	}

	return nil
}
