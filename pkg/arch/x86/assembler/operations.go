package assembler

import (
	"errors"
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/x86/parser"
	isa "github.com/retroenv/retrogolib/arch/cpu/x86"
)

var impliedOpcodes = map[string]byte{
	isa.NopName: 0x90, isa.RetName: 0xc3, isa.IretName: 0xcf, isa.CbwName: 0x98, isa.CwdName: 0x99,
	isa.ClcName: 0xf8, isa.StcName: 0xf9, isa.CmcName: 0xf5, isa.CldName: 0xfc, isa.StdName: 0xfd,
	isa.MovsbName: 0xa4, isa.MovswName: 0xa5, isa.CmpsbName: 0xa6, isa.CmpswName: 0xa7,
	isa.StosbName: 0xaa, isa.StoswName: 0xab, isa.LodsbName: 0xac, isa.LodswName: 0xad,
	isa.ScasbName: 0xae, isa.ScaswName: 0xaf,
}

var unaryFields = map[string]byte{isa.NotName: 2, isa.NegName: 3, isa.MulName: 4, isa.ImulName: 5, isa.DivName: 6, isa.IdivName: 7}

var shiftFields = map[string]byte{isa.RolName: 0, isa.RorName: 1, isa.RclName: 2, isa.RcrName: 3, isa.ShlName: 4, isa.ShrName: 5, isa.SarName: 7}

func stackInstruction(assigner arch.AddressAssigner, name string, operand any, resolve bool) ([]byte, error) {
	pop := name == isa.PopName
	if register, valid := parser.RegisterValue(operand); valid {
		return stackRegister(register, pop)
	}
	opcode := byte(0xff)
	field := byte(6)
	if pop {
		opcode = 0x8f
		field = 0
	}

	return encodeRM(assigner, opcode, field, operand, 2, resolve)
}

func leaInstruction(assigner arch.AddressAssigner, args []any, resolve bool) ([]byte, error) {
	destination, valid := parser.RegisterValue(args[0])
	if !valid || !destination.Is16Bit() {
		return nil, errors.New("8086 LEA destination needs a word register")
	}
	memory, valid := args[1].(parser.MemoryOperand)
	if !valid || memory.Segment != 0 {
		return nil, errors.New("8086 LEA needs memory without a segment override")
	}
	return encodeRM(assigner, 0x8d, registerCode(destination), memory, 2, resolve)
}

func unaryInstruction(assigner arch.AddressAssigner, name string, operand any, resolve bool) ([]byte, error) {
	width := operandWidth(operand)
	opcode := byte(0xf6)
	field := unaryFields[name]
	if name == isa.IncName || name == isa.DecName {
		field = 0
		if name == isa.DecName {
			field = 1
		}

		if register, valid := parser.RegisterValue(operand); valid && register.Is16Bit() {
			return []byte{0x40 + field*8 + registerCode(register)}, nil
		}
		opcode = 0xfe
	}

	if width == 2 {
		opcode++
	}

	return encodeRM(assigner, opcode, field, operand, width, resolve)
}

func shiftInstruction(assigner arch.AddressAssigner, name string, args []any, resolve bool) ([]byte, error) {
	opcode := byte(0xd0)
	if count, valid := parser.RegisterValue(args[1]); valid {
		if count != isa.RegCL {
			return nil, errors.New("8086 variable shift count must be CL")
		}
		opcode = 0xd2
	} else {
		value, literal := args[1].(uint64)
		if !literal || value != 1 {
			return nil, errors.New("8086 immediate shift count must be one")
		}
	}

	width := operandWidth(args[0])
	if width == 2 {
		opcode++
	}

	return encodeRM(assigner, opcode, shiftFields[name], args[0], width, resolve)
}

func repeatInstruction(name string, operand any) ([]byte, error) {
	body, valid := operand.(parser.RepeatedString)
	if !valid || !isa.RepeatableInstructions.Contains(string(body)) {
		return nil, errors.New("8086 repeat prefix needs a supported string instruction")
	}
	opcode, supported := impliedOpcodes[string(body)]
	if !supported {
		return nil, fmt.Errorf("unsupported 8086 repeated operation %s", body)
	}
	prefix := byte(0xf3)
	if name == isa.RepnzName {
		if opcode != 0xa6 && opcode != 0xa7 && opcode != 0xae && opcode != 0xaf {
			return nil, errors.New("8086 REPNZ needs a compare or scan string instruction")
		}
		prefix = 0xf2
	}

	return []byte{prefix, opcode}, nil
}

func stackRegister(register isa.RegisterParam, pop bool) ([]byte, error) {
	if register.Is16Bit() {
		opcode := byte(0x50)
		if pop {
			opcode = 0x58
		}

		return []byte{opcode + registerCode(register)}, nil
	}
	if register.IsSegment() && (!pop || register != isa.RegCS) {
		opcode := byte(0x06 + 8*(register-isa.RegES))
		if pop {
			opcode++
		}

		return []byte{opcode}, nil
	}
	return nil, errors.New("8086 stack operand needs a word register or a permitted segment register")
}
