package assembler

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/retroenv/retroasm/pkg/arch"
	sm83parser "github.com/retroenv/retroasm/pkg/arch/sm83/parser"
	cpusm83 "github.com/retroenv/retrogolib/arch/cpu/sm83"
)

var (
	errMissingOperand         = errors.New("missing operand value")
	errInvalidBitNumber       = errors.New("invalid bit number")
	errUnsupportedBitRegister = errors.New("unsupported bit register parameter")
	errUnsupportedAddressing  = errors.New("unsupported addressing mode")
)

// GenerateInstructionOpcode generates SM83 opcode bytes for an already resolved instruction.
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
	resolved sm83parser.ResolvedInstruction,
	opcodeInfo cpusm83.OpcodeInfo,
	addressing cpusm83.AddressingMode,
) ([]byte, error) {

	if isCBBitInstruction(resolved.Instruction) {
		return buildBitOpcode(assigner, resolved, opcodeInfo)
	}

	opcodes := baseOpcodeBytes(opcodeInfo)

	switch addressing {
	case cpusm83.ImpliedAddressing, cpusm83.RegisterAddressing, cpusm83.BitAddressing:
		return opcodes, nil

	case cpusm83.ImmediateAddressing:
		return appendImmediateOperand(assigner, resolved, opcodeInfo, opcodes)

	case cpusm83.ExtendedAddressing:
		return appendExtendedOperand(assigner, resolved, opcodes)

	case cpusm83.RelativeAddressing:
		return appendRelativeOperand(assigner, ins, resolved, opcodeInfo, opcodes)

	case cpusm83.RegisterIndirectAddressing:
		return appendOptionalByteOperand(assigner, resolved, opcodeInfo, opcodes)

	default:
		return nil, fmt.Errorf("%w: %d", errUnsupportedAddressing, addressing)
	}
}

func appendImmediateOperand(
	assigner arch.AddressAssigner,
	resolved sm83parser.ResolvedInstruction,
	opcodeInfo cpusm83.OpcodeInfo,
	opcodes []byte,
) ([]byte, error) {

	remaining := int(opcodeInfo.Size) - len(opcodes)
	switch remaining {
	case 0:
		return opcodes, nil

	case 1:
		value, err := resolvedOperandValue(assigner, resolved)
		if err != nil {
			return nil, err
		}
		if value > math.MaxUint8 {
			return nil, fmt.Errorf("immediate value %d exceeds byte", value)
		}
		return append(opcodes, byte(value)), nil

	case 2:
		value, err := resolvedOperandValue(assigner, resolved)
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

func appendExtendedOperand(assigner arch.AddressAssigner, resolved sm83parser.ResolvedInstruction, opcodes []byte) ([]byte, error) {
	value, err := resolvedOperandValue(assigner, resolved)
	if err != nil {
		return nil, err
	}
	if value > math.MaxUint16 {
		return nil, fmt.Errorf("extended address %d exceeds word", value)
	}
	return binary.LittleEndian.AppendUint16(opcodes, uint16(value)), nil
}

func appendRelativeOperand(
	assigner arch.AddressAssigner,
	ins arch.Instruction,
	resolved sm83parser.ResolvedInstruction,
	opcodeInfo cpusm83.OpcodeInfo,
	opcodes []byte,
) ([]byte, error) {

	value, err := resolvedOperandValue(assigner, resolved)
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

func appendOptionalByteOperand(
	assigner arch.AddressAssigner,
	resolved sm83parser.ResolvedInstruction,
	opcodeInfo cpusm83.OpcodeInfo,
	opcodes []byte,
) ([]byte, error) {

	remaining := int(opcodeInfo.Size) - len(opcodes)
	if remaining == 0 {
		return opcodes, nil
	}
	if remaining != 1 {
		return nil, fmt.Errorf("%w: expected 0 or 1 operand byte, got %d", errUnsupportedAddressing, remaining)
	}

	value, err := resolvedOperandValue(assigner, resolved)
	if err != nil {
		return nil, err
	}
	if value > math.MaxUint8 {
		return nil, fmt.Errorf("operand value %d exceeds byte", value)
	}
	return append(opcodes, byte(value)), nil
}

func baseOpcodeBytes(opcodeInfo cpusm83.OpcodeInfo) []byte {
	opcodes := make([]byte, 0, opcodeInfo.Size)
	if opcodeInfo.Prefix != 0 {
		opcodes = append(opcodes, opcodeInfo.Prefix)
	}
	return append(opcodes, opcodeInfo.Opcode)
}

func buildBitOpcode(assigner arch.AddressAssigner, resolved sm83parser.ResolvedInstruction, opcodeInfo cpusm83.OpcodeInfo) ([]byte, error) {
	opcodes := make([]byte, 0, 2)
	opcodes = append(opcodes, opcodeInfo.Prefix)

	bitNumber, err := resolvedOperandValue(assigner, resolved)
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

func resolvedOperandValue(assigner arch.AddressAssigner, resolved sm83parser.ResolvedInstruction) (uint64, error) {
	if len(resolved.OperandValues) == 0 {
		return 0, fmt.Errorf("%w: operand index 0", errMissingOperand)
	}

	value, err := assigner.ArgumentValue(resolved.OperandValues[0])
	if err != nil {
		return 0, fmt.Errorf("resolving operand value: %w", err)
	}
	return value, nil
}

// bitNumberShift is the bit position shift for encoding the bit number
// in CB instructions (bit number occupies bits 5-3).
const bitNumberShift = 3

var bitRegisterCodes = map[cpusm83.RegisterParam]byte{
	cpusm83.RegB:          0,
	cpusm83.RegC:          1,
	cpusm83.RegD:          2,
	cpusm83.RegE:          3,
	cpusm83.RegH:          4,
	cpusm83.RegL:          5,
	cpusm83.RegHLIndirect: 6,
	cpusm83.RegA:          7,
}

func bitRegisterCode(resolved sm83parser.ResolvedInstruction) (byte, error) {
	target := cpusm83.RegHLIndirect
	if len(resolved.RegisterParams) > 0 {
		target = resolved.RegisterParams[len(resolved.RegisterParams)-1]
	}

	code, ok := bitRegisterCodes[target]
	if !ok {
		return 0, fmt.Errorf("%w: %s", errUnsupportedBitRegister, target.String())
	}
	return code, nil
}

func isCBBitInstruction(instruction *cpusm83.Instruction) bool {
	return instruction == cpusm83.CBBit || instruction == cpusm83.CBRes || instruction == cpusm83.CBSet
}
