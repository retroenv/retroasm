package assembler

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/retroenv/retroasm/pkg/arch"
	z80parser "github.com/retroenv/retroasm/pkg/arch/z80/parser"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
)

var (
	errMissingOperand         = errors.New("missing operand value")
	errInvalidBitNumber       = errors.New("invalid bit number")
	errUnsupportedBitRegister = errors.New("unsupported bit register parameter")
	errUnsupportedAddressing  = errors.New("unsupported addressing mode")
)

// GenerateInstructionOpcode generates Z80 opcode bytes for an already resolved instruction.
func GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	resolved, err := resolvedInstruction(ins.Argument())
	if err != nil {
		return fmt.Errorf("resolving instruction argument: %w", err)
	}

	opcodeInfo, addressing, err := opcodeInfoForResolvedInstruction(resolved)
	if err != nil {
		return fmt.Errorf("resolving opcode info for '%s': %w", ins.Name(), err)
	}

	opcodes, err := buildOpcodeBytes(assigner, ins, resolved, opcodeInfo, addressing)
	if err != nil {
		return fmt.Errorf("building opcode bytes for '%s': %w", ins.Name(), err)
	}

	ins.SetAddressing(int(addressing))
	ins.SetOpcodes(opcodes)
	ins.SetSize(len(opcodes))

	return nil
}

func buildOpcodeBytes(
	assigner arch.AddressAssigner,
	ins arch.Instruction,
	resolved z80parser.ResolvedInstruction,
	opcodeInfo cpuz80.OpcodeInfo,
	addressing cpuz80.AddressingMode,
) ([]byte, error) {

	if isIndexedBitInstruction(resolved.Instruction) {
		return buildIndexedBitOpcode(assigner, resolved, opcodeInfo)
	}

	if isCBBitInstruction(resolved.Instruction) {
		return buildBitOpcode(assigner, resolved, opcodeInfo)
	}

	opcodes := baseOpcodeBytes(opcodeInfo)

	switch addressing {
	case cpuz80.ImpliedAddressing, cpuz80.RegisterAddressing, cpuz80.BitAddressing:
		return opcodes, nil

	case cpuz80.ImmediateAddressing:
		return appendImmediateOperand(assigner, resolved, opcodeInfo, opcodes)

	case cpuz80.ExtendedAddressing:
		return appendExtendedOperand(assigner, resolved, opcodes)

	case cpuz80.RelativeAddressing:
		return appendRelativeOperand(assigner, ins, resolved, opcodeInfo, opcodes)

	case cpuz80.RegisterIndirectAddressing, cpuz80.PortAddressing:
		return appendOptionalByteOperand(assigner, resolved, opcodeInfo, opcodes)

	default:
		return nil, fmt.Errorf("%w: %d", errUnsupportedAddressing, addressing)
	}
}

func appendImmediateOperand(
	assigner arch.AddressAssigner,
	resolved z80parser.ResolvedInstruction,
	opcodeInfo cpuz80.OpcodeInfo,
	opcodes []byte,
) ([]byte, error) {

	remaining := int(opcodeInfo.Size) - len(opcodes)
	switch remaining {
	case 0:
		return opcodes, nil

	case 1:
		value, err := resolvedOperandValue(assigner, resolved, 0)
		if err != nil {
			return nil, err
		}
		if value > math.MaxUint8 {
			return nil, fmt.Errorf("immediate value %d exceeds byte", value)
		}
		return append(opcodes, byte(value)), nil

	case 2:
		// Two separate byte operands (e.g., LD (IX+d),n: displacement + immediate).
		if len(resolved.OperandValues) >= 2 {
			for i := range 2 {
				v, err := resolvedOperandValue(assigner, resolved, i)
				if err != nil {
					return nil, err
				}
				if v > math.MaxUint8 {
					return nil, fmt.Errorf("immediate byte %d value %d exceeds byte", i, v)
				}
				opcodes = append(opcodes, byte(v))
			}
			return opcodes, nil
		}

		value, err := resolvedOperandValue(assigner, resolved, 0)
		if err != nil {
			return nil, err
		}
		if value > math.MaxUint16 {
			return nil, fmt.Errorf("immediate value %d exceeds word", value)
		}
		return binary.LittleEndian.AppendUint16(opcodes, uint16(value)), nil

	default:
		return nil, fmt.Errorf("%w: immediate operand byte width %d", errUnsupportedAddressing, remaining)
	}
}

func appendExtendedOperand(assigner arch.AddressAssigner, resolved z80parser.ResolvedInstruction, opcodes []byte) ([]byte, error) {
	value, err := resolvedOperandValue(assigner, resolved, 0)
	if err != nil {
		return nil, err
	}
	if value > math.MaxUint16 {
		return nil, fmt.Errorf("extended address %d exceeds word", value)
	}
	return binary.LittleEndian.AppendUint16(opcodes, uint16(value)), nil
}

func appendOptionalByteOperand(
	assigner arch.AddressAssigner,
	resolved z80parser.ResolvedInstruction,
	opcodeInfo cpuz80.OpcodeInfo,
	opcodes []byte,
) ([]byte, error) {

	remaining := int(opcodeInfo.Size) - len(opcodes)
	if remaining == 0 {
		return opcodes, nil
	}
	if remaining != 1 {
		return nil, fmt.Errorf("%w: expected 0 or 1 operand byte, got %d", errUnsupportedAddressing, remaining)
	}

	value, err := resolvedOperandValue(assigner, resolved, 0)
	if err != nil {
		return nil, err
	}
	if value > math.MaxUint8 {
		return nil, fmt.Errorf("operand value %d exceeds byte", value)
	}
	return append(opcodes, byte(value)), nil
}

func appendRelativeOperand(
	assigner arch.AddressAssigner,
	ins arch.Instruction,
	resolved z80parser.ResolvedInstruction,
	opcodeInfo cpuz80.OpcodeInfo,
	opcodes []byte,
) ([]byte, error) {

	value, err := resolvedOperandValue(assigner, resolved, 0)
	if err != nil {
		return nil, err
	}

	addressAfterInstruction := ins.Address() + uint64(opcodeInfo.Size)
	offset, err := assigner.RelativeOffset(value, addressAfterInstruction)
	if err != nil {
		return nil, fmt.Errorf("resolving relative offset: %w", err)
	}
	return append(opcodes, offset), nil
}

func baseOpcodeBytes(opcodeInfo cpuz80.OpcodeInfo) []byte {
	opcodes := make([]byte, 0, opcodeInfo.Size)
	if opcodeInfo.Prefix != 0 {
		opcodes = append(opcodes, opcodeInfo.Prefix)
	}
	return append(opcodes, opcodeInfo.Opcode)
}

func buildBitOpcode(assigner arch.AddressAssigner, resolved z80parser.ResolvedInstruction, opcodeInfo cpuz80.OpcodeInfo) ([]byte, error) {
	opcodes := make([]byte, 0, 2)
	opcodes = append(opcodes, opcodeInfo.Prefix)

	bitNumber, err := resolvedOperandValue(assigner, resolved, 0)
	if err != nil {
		return nil, err
	}
	if bitNumber > 7 {
		return nil, fmt.Errorf("%w: %d", errInvalidBitNumber, bitNumber)
	}

	registerCode, err := bitRegisterCode(resolved)
	if err != nil {
		return nil, err
	}

	opcode := opcodeInfo.Opcode + byte(bitNumber<<bitNumberShift) + registerCode
	opcodes = append(opcodes, opcode)
	return opcodes, nil
}

func buildIndexedBitOpcode(
	assigner arch.AddressAssigner,
	resolved z80parser.ResolvedInstruction,
	opcodeInfo cpuz80.OpcodeInfo,
) ([]byte, error) {

	displacementIndex := 0
	bitNumber := uint64(0)

	switch resolved.Instruction {
	case cpuz80.DdcbBit, cpuz80.DdcbRes, cpuz80.DdcbSet, cpuz80.FdcbBit, cpuz80.FdcbRes, cpuz80.FdcbSet:
		value, err := resolvedOperandValue(assigner, resolved, 0)
		if err != nil {
			return nil, err
		}
		if value > 7 {
			return nil, fmt.Errorf("%w: %d", errInvalidBitNumber, value)
		}
		bitNumber = value
		displacementIndex = 1
	}

	displacement, err := resolvedOperandValue(assigner, resolved, displacementIndex)
	if err != nil {
		return nil, err
	}
	if displacement > math.MaxUint8 {
		return nil, fmt.Errorf("indexed displacement %d exceeds byte", displacement)
	}

	registerCode, err := bitRegisterCode(resolved)
	if err != nil {
		return nil, err
	}

	opcode := opcodeInfo.Opcode + byte(bitNumber<<bitNumberShift) + registerCode
	return []byte{opcodeInfo.Prefix, cpuz80.PrefixCB, byte(displacement), opcode}, nil
}

func resolvedOperandValue(assigner arch.AddressAssigner, resolved z80parser.ResolvedInstruction, index int) (uint64, error) {
	if index < 0 || index >= len(resolved.OperandValues) {
		return 0, fmt.Errorf("%w: operand index %d", errMissingOperand, index)
	}

	value, err := assigner.ArgumentValue(resolved.OperandValues[index])
	if err != nil {
		return 0, fmt.Errorf("resolving operand %d value: %w", index, err)
	}
	return value, nil
}

// bitNumberShift is the bit position shift for encoding the bit number
// in CB/DDCB/FDCB instructions (bit number occupies bits 5-3).
const bitNumberShift = 3

var bitRegisterCodes = map[cpuz80.RegisterParam]byte{
	cpuz80.RegB:          0,
	cpuz80.RegC:          1,
	cpuz80.RegD:          2,
	cpuz80.RegE:          3,
	cpuz80.RegH:          4,
	cpuz80.RegL:          5,
	cpuz80.RegHLIndirect: 6,
	cpuz80.RegA:          7,
}

func bitRegisterCode(resolved z80parser.ResolvedInstruction) (byte, error) {
	target := cpuz80.RegHLIndirect
	if len(resolved.RegisterParams) > 0 {
		target = resolved.RegisterParams[len(resolved.RegisterParams)-1]
	}

	code, ok := bitRegisterCodes[target]
	if !ok {
		return 0, fmt.Errorf("%w: %s", errUnsupportedBitRegister, target.String())
	}
	return code, nil
}

func isCBBitInstruction(instruction *cpuz80.Instruction) bool {
	return instruction == cpuz80.CBBit || instruction == cpuz80.CBRes || instruction == cpuz80.CBSet
}

func isIndexedBitInstruction(instruction *cpuz80.Instruction) bool {
	switch instruction {
	case cpuz80.DdcbShift, cpuz80.DdcbBit, cpuz80.DdcbRes, cpuz80.DdcbSet,
		cpuz80.FdcbShift, cpuz80.FdcbBit, cpuz80.FdcbRes, cpuz80.FdcbSet:
		return true
	default:
		return false
	}
}
