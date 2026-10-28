package assembler

import (
	"errors"
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/x86/parser"
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
	if conditionalOpcode(instruction.Name()) != 0 && instruction.Size() == 5 {
		return longConditional(assigner, instruction.Name(), instruction.Argument(), instruction.Address(), resolve)
	}
	return encodeForm(assigner, instruction.Name(), instruction.Argument(), instruction.Address(), resolve)
}

func encodeForm(assigner arch.AddressAssigner, mnemonic string, argument any, address uint64,
	resolve bool) ([]byte, error) {

	encoded, err := selectEncoding(assigner, mnemonic, argument, address, resolve)
	if err != nil {
		return nil, err
	}
	if err := validate8086Encoding(encoded); err != nil {
		return nil, err
	}
	return encoded, nil
}

// validate8086Encoding checks the selected opcode generation after legacy
// prefixes. Immediate and displacement bytes are not instruction prefixes.
func validate8086Encoding(encoded []byte) error {
	for index, value := range encoded {
		switch value {
		case 0x26, 0x2e, 0x36, 0x3e, 0xf0, 0xf2, 0xf3:
			continue
		}

		opcode := uint16(value)
		if value == 0x0f {
			if index+1 >= len(encoded) {
				return errors.New("incomplete x86 escape opcode")
			}
			opcode = opcode<<8 | uint16(encoded[index+1])
		}

		info := isa.OpcodeInfo{
			Opcode: opcode,
			Size:   1,
		}
		if !isa.CPU8086.Supports(info) {
			return fmt.Errorf("opcode %#x needs CPU %s or an explicit classification; target is 8086", opcode, info.MinimumCPU())
		}
		return nil
	}

	return errors.New("empty x86 encoding or incomplete prefix sequence")
}

func selectEncoding(assigner arch.AddressAssigner, mnemonic string, argument any, address uint64,
	resolve bool) ([]byte, error) {

	name := strings.ToLower(mnemonic)
	var args []any
	if list, ok := argument.([]any); ok {
		args = list
	} else if argument != nil {
		args = []any{argument}
	}

	if opcode, known := impliedOpcodes[name]; known && len(args) == 0 {
		return []byte{opcode}, nil
	}
	switch len(args) {
	case 1:
		return singleOperandInstruction(assigner, name, args[0], address, resolve)
	case 2:
		return pairedOperandInstruction(assigner, name, args, resolve)
	default:
		return nil, fmt.Errorf("unsupported 8086 instruction or operands: %s", mnemonic)
	}
}

func singleOperandInstruction(assigner arch.AddressAssigner, name string, operand any, address uint64,
	resolve bool) ([]byte, error) {

	if conditionalOpcode(name) != 0 || name == isa.CallName || name == isa.JmpName {
		return controlInstruction(assigner, name, []any{operand}, address, resolve)
	}
	switch name {
	case isa.IntName:
		value, err := immediate(assigner, operand, 1, resolve)
		return []byte{0xcd, byte(value)}, err

	case isa.RetName:
		value, err := immediate(assigner, operand, 2, resolve)
		return []byte{0xc2, byte(value), byte(value >> 8)}, err

	case isa.PushName, isa.PopName:
		return stackInstruction(assigner, name, operand, resolve)
	case isa.IncName, isa.DecName, isa.NegName, isa.NotName, isa.MulName, isa.ImulName, isa.DivName, isa.IdivName:
		return unaryInstruction(assigner, name, operand, resolve)
	case isa.RepName, isa.RepzName, isa.RepnzName:
		return repeatInstruction(name, operand)
	default:
		return nil, fmt.Errorf("unsupported 8086 instruction or operands: %s", name)
	}
}

func pairedOperandInstruction(assigner arch.AddressAssigner, name string, args []any, resolve bool) ([]byte, error) {
	if _, known := binaryOpcodes[name]; known {
		return binaryInstruction(assigner, name, args, resolve)
	}
	switch name {
	case isa.LeaName:
		return leaInstruction(assigner, args, resolve)
	case isa.ShlName, isa.ShrName, isa.SarName, isa.RolName, isa.RorName, isa.RclName, isa.RcrName:
		return shiftInstruction(assigner, name, args, resolve)
	default:
		return nil, fmt.Errorf("unsupported 8086 instruction or operands: %s", name)
	}
}

func immediate(assigner arch.AddressAssigner, arg any, width int, resolve bool) (uint64, error) {
	if value, signed := arg.(parser.SignedImmediate); signed {
		if int64(value) < -(1<<(width*8-1)) || int64(value) > (1<<(width*8))-1 {
			return 0, fmt.Errorf("signed 8086 immediate %d exceeds %d bytes", value, width)
		}
		return uint64(value) & ((1 << (width * 8)) - 1), nil
	}
	switch arg.(type) {
	case parser.RegisterOperand, parser.MemoryOperand, parser.RepeatedString:
		return 0, errors.New("8086 immediate must be a value or symbol")
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
