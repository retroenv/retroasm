package assembler

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/x86/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	isa "github.com/retroenv/retrogolib/arch/cpu/x86"
)

func operandWidth(value any) int {
	if register, valid := parser.RegisterValue(value); valid {
		return register.GetRegisterSize()
	}
	if memory, valid := value.(parser.MemoryOperand); valid {
		return int(memory.Width)
	}
	return 0
}

func registerCode(register isa.RegisterParam) byte {
	if register.Is16Bit() {
		return byte(register - isa.RegAX)
	}
	return byte(register)
}

func encodeRM(assigner arch.AddressAssigner, opcode, field byte, operand any, width int, resolve bool) ([]byte, error) {
	if width != 1 && width != 2 {
		return nil, errors.New("8086 operand needs byte or word width")
	}
	if register, valid := parser.RegisterValue(operand); valid {
		if !register.Is8Bit() && !register.Is16Bit() || register.GetRegisterSize() != width {
			return nil, errors.New("8086 register width does not match the operation")
		}
		return []byte{opcode, isa.NewModRM(3, field, registerCode(register)).ToByte()}, nil
	}
	memory, valid := operand.(parser.MemoryOperand)
	if !valid {
		return nil, errors.New("8086 operand must be a register or memory")
	}
	if memory.Width != 0 && int(memory.Width) != width {
		return nil, errors.New("8086 memory width does not match the operation")
	}
	prefix, err := segmentPrefix(memory.Segment)
	if err != nil {
		return nil, err
	}
	mode, rm, displacementBytes, err := memoryForm(memory)
	if err != nil {
		return nil, err
	}
	displacement, err := memoryDisplacement(assigner, memory, resolve)
	if err != nil {
		return nil, err
	}
	encoded := prefix
	encoded = append(encoded, opcode, isa.NewModRM(mode, field, rm).ToByte())
	switch displacementBytes {
	case 1:
		encoded = append(encoded, byte(displacement))
	case 2:
		encoded = binary.LittleEndian.AppendUint16(encoded, uint16(displacement))
	}

	return encoded, nil
}

func segmentPrefix(segment isa.RegisterParam) ([]byte, error) {
	if segment == 0 {
		return nil, nil
	}
	if !segment.IsSegment() {
		return nil, fmt.Errorf("invalid 8086 segment register %v", segment)
	}
	return []byte{0x26 + 8*byte(segment-isa.RegES)}, nil
}

func memoryForm(memory parser.MemoryOperand) (mode, rm byte, displacementBytes int, err error) {
	if memory.Displacement < -32768 || memory.Displacement > 65535 {
		return 0, 0, 0, errors.New("8086 memory displacement exceeds one word")
	}
	if memory.Address == isa.RegMem {
		return 0, 6, 2, nil
	}
	if memory.Address < isa.RegBXSIRef || memory.Address > isa.RegBXRef {
		return 0, 0, 0, errors.New("invalid 8086 effective address")
	}
	rm = byte(memory.Address - isa.RegBXSIRef)
	displacement := int64(int16(uint16(memory.Displacement)))
	if memory.Symbol != "" || displacement < -128 || displacement > 127 {
		return 2, rm, 2, nil
	}
	if displacement != 0 || memory.Address == isa.RegBPRef {
		return 1, rm, 1, nil
	}
	return 0, rm, 0, nil
}

func memoryDisplacement(assigner arch.AddressAssigner, memory parser.MemoryOperand, resolve bool) (int64, error) {
	displacement := memory.Displacement
	if memory.Symbol != "" && resolve {
		value, err := assigner.ArgumentValue(ast.NewLabel(memory.Symbol))
		if err != nil {
			return 0, fmt.Errorf("resolving 8086 memory symbol: %w", err)
		}
		if value > 65535 {
			return 0, errors.New("8086 memory symbol exceeds a segment")
		}
		displacement += int64(value)
	}

	if displacement < -32768 || displacement > 65535 || memory.Address == isa.RegMem && displacement < 0 && (memory.Symbol == "" || resolve) {
		return 0, errors.New("8086 effective displacement exceeds one word")
	}
	return displacement, nil
}
