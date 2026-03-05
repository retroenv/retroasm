package assembler

import (
	"fmt"
	"slices"
	"testing"

	z80parser "github.com/retroenv/retroasm/pkg/arch/z80/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
	"github.com/retroenv/retrogolib/assert"
	"github.com/retroenv/retrogolib/set"
)

const coverageProgramCounter = uint64(0x8000)

func TestOpcodeCoverage_AllInstructionVariants(t *testing.T) {
	instructions := allInstructionVariantsForCoverage()
	assert.NotEmpty(t, instructions)

	for index, instruction := range instructions {
		resolved, opcodeInfo, err := coverageResolvedInstruction(instruction)
		assert.NoError(t, err)

		t.Run(fmt.Sprintf("%03d_%s", index, instruction.Name), func(t *testing.T) {
			assigner := &mockAssigner{pc: coverageProgramCounter}
			ins := &mockInstruction{
				name:     instruction.Name,
				argument: resolved,
			}

			nextPC, err := AssignInstructionAddress(assigner, ins)
			assert.NoError(t, err)
			assert.Equal(t, coverageProgramCounter, ins.Address())
			assert.Equal(t, int(opcodeInfo.Size), ins.Size())
			assert.Equal(t, coverageProgramCounter+uint64(opcodeInfo.Size), nextPC)

			err = GenerateInstructionOpcode(assigner, ins)
			assert.NoError(
				t,
				err,
				"instruction=%s addressing=%d prefix=%02x opcode=%02x size=%d regs=%v operands=%v",
				instruction.Name,
				resolved.Addressing,
				opcodeInfo.Prefix,
				opcodeInfo.Opcode,
				opcodeInfo.Size,
				resolved.RegisterParams,
				resolved.OperandValues,
			)
			assert.Len(t, ins.Opcodes(), int(opcodeInfo.Size))
			assert.Equal(t, int(resolved.Addressing), ins.Addressing())
		})
	}
}

func allInstructionVariantsForCoverage() []*cpuz80.Instruction {
	seen := set.New[*cpuz80.Instruction]()
	instructions := make([]*cpuz80.Instruction, 0, 256)

	instructions = addInstructionSlice(instructions, seen, tableInstructionSlice(cpuz80.Opcodes))
	instructions = addInstructionSlice(instructions, seen, tableInstructionSlice(cpuz80.EDOpcodes))
	instructions = addInstructionSlice(instructions, seen, tableInstructionSlice(cpuz80.DDOpcodes))
	instructions = addInstructionSlice(instructions, seen, tableInstructionSlice(cpuz80.FDOpcodes))

	instructions = addInstructionSlice(instructions, seen, []*cpuz80.Instruction{
		cpuz80.CBRlc,
		cpuz80.CBRrc,
		cpuz80.CBRl,
		cpuz80.CBRr,
		cpuz80.CBSla,
		cpuz80.CBSra,
		cpuz80.CBSll,
		cpuz80.CBSrl,
		cpuz80.CBBit,
		cpuz80.CBRes,
		cpuz80.CBSet,
		cpuz80.DdcbShift,
		cpuz80.DdcbBit,
		cpuz80.DdcbRes,
		cpuz80.DdcbSet,
		cpuz80.FdcbShift,
		cpuz80.FdcbBit,
		cpuz80.FdcbRes,
		cpuz80.FdcbSet,
	})

	return instructions
}

func addInstructionSlice(
	instructions []*cpuz80.Instruction,
	seen set.Set[*cpuz80.Instruction],
	candidates []*cpuz80.Instruction,
) []*cpuz80.Instruction {

	for _, instruction := range candidates {
		if instruction == nil {
			continue
		}
		if seen.Contains(instruction) {
			continue
		}
		// Skip undocumented alias instructions that have no addressing modes
		// or register opcodes — they exist only for emulator decoding and
		// cannot be assembled.
		if instruction.Unofficial && len(instruction.Addressing) == 0 &&
			len(instruction.RegisterOpcodes) == 0 && len(instruction.RegisterPairOpcodes) == 0 {

			continue
		}
		seen.Add(instruction)
		instructions = append(instructions, instruction)
	}
	return instructions
}

func tableInstructionSlice(table [256]cpuz80.Opcode) []*cpuz80.Instruction {
	instructions := make([]*cpuz80.Instruction, 0, len(table))
	for _, opcode := range table {
		if opcode.Instruction == nil {
			continue
		}
		instructions = append(instructions, opcode.Instruction)
	}
	return instructions
}

func coverageResolvedInstruction(instruction *cpuz80.Instruction) (z80parser.ResolvedInstruction, cpuz80.OpcodeInfo, error) {
	resolved := z80parser.ResolvedInstruction{
		Instruction: instruction,
	}

	registerCombos := registerParamCombinations(instruction)
	addressings := addressingCandidates(instruction)

	for _, registers := range registerCombos {
		resolved.RegisterParams = registers

		for _, addressing := range addressings {
			resolved.Addressing = addressing

			opcodeInfo, resolvedAddressing, err := opcodeInfoForResolvedInstruction(resolved)
			if err != nil {
				continue
			}

			resolved.Addressing = resolvedAddressing
			resolved.OperandValues = nil

			err = setCoverageOperandValues(&resolved, opcodeInfo)
			if err != nil {
				return z80parser.ResolvedInstruction{}, cpuz80.OpcodeInfo{}, err
			}

			return resolved, opcodeInfo, nil
		}
	}

	return z80parser.ResolvedInstruction{}, cpuz80.OpcodeInfo{}, fmt.Errorf("no opcode mapping for instruction %q", instruction.Name)
}

func registerParamCombinations(instruction *cpuz80.Instruction) [][]cpuz80.RegisterParam {
	combinations := make([][]cpuz80.RegisterParam, 0, len(instruction.RegisterOpcodes)+len(instruction.RegisterPairOpcodes)+1)

	if len(instruction.RegisterPairOpcodes) > 0 {
		pairs := make([][2]cpuz80.RegisterParam, 0, len(instruction.RegisterPairOpcodes))
		for pair := range instruction.RegisterPairOpcodes {
			pairs = append(pairs, pair)
		}
		slices.SortFunc(pairs, compareRegisterPair)
		for _, pair := range pairs {
			combinations = append(combinations, []cpuz80.RegisterParam{pair[0], pair[1]})
		}
	}

	if len(instruction.RegisterOpcodes) > 0 {
		registers := make([]cpuz80.RegisterParam, 0, len(instruction.RegisterOpcodes))
		for register := range instruction.RegisterOpcodes {
			registers = append(registers, register)
		}
		slices.Sort(registers)
		for _, register := range registers {
			combinations = append(combinations, []cpuz80.RegisterParam{register})
		}
	}

	return append(combinations, nil)
}

func compareRegisterPair(a, b [2]cpuz80.RegisterParam) int {
	if a[0] < b[0] {
		return -1
	}
	if a[0] > b[0] {
		return 1
	}
	if a[1] < b[1] {
		return -1
	}
	if a[1] > b[1] {
		return 1
	}
	return 0
}

func addressingCandidates(instruction *cpuz80.Instruction) []cpuz80.AddressingMode {
	if len(instruction.Addressing) == 0 {
		return []cpuz80.AddressingMode{cpuz80.NoAddressing}
	}

	addressings := make([]cpuz80.AddressingMode, 0, len(instruction.Addressing)+1)
	for mode := range instruction.Addressing {
		addressings = append(addressings, mode)
	}
	slices.Sort(addressings)

	if !slices.Contains(addressings, cpuz80.NoAddressing) {
		addressings = append(addressings, cpuz80.NoAddressing)
	}

	return addressings
}

func setCoverageOperandValues(resolved *z80parser.ResolvedInstruction, opcodeInfo cpuz80.OpcodeInfo) error {
	if isIndexedBitInstruction(resolved.Instruction) {
		return setIndexedBitOperands(resolved)
	}

	if isCBBitInstruction(resolved.Instruction) {
		resolved.OperandValues = []ast.Node{ast.NewNumber(3)}
		return nil
	}

	switch resolved.Addressing {
	case cpuz80.ImmediateAddressing:
		return setImmediateOperands(resolved, opcodeInfo)

	case cpuz80.ExtendedAddressing:
		resolved.OperandValues = []ast.Node{ast.NewNumber(0x1234)}
		return nil

	case cpuz80.RelativeAddressing:
		resolved.OperandValues = []ast.Node{ast.NewNumber(coverageProgramCounter + 4)}
		return nil

	case cpuz80.RegisterIndirectAddressing, cpuz80.PortAddressing:
		return setOptionalByteOperand(resolved, opcodeInfo)

	default:
		return nil
	}
}

func setImmediateOperands(resolved *z80parser.ResolvedInstruction, opcodeInfo cpuz80.OpcodeInfo) error {
	operandWidth := int(opcodeInfo.Size) - baseOpcodeWidth(opcodeInfo)
	switch operandWidth {
	case 0:
		return nil
	case 1:
		resolved.OperandValues = []ast.Node{ast.NewNumber(0x12)}
		return nil
	case 2:
		resolved.OperandValues = []ast.Node{ast.NewNumber(0x1234)}
		return nil
	default:
		return fmt.Errorf("unsupported immediate operand width %d for instruction %q", operandWidth, resolved.Instruction.Name)
	}
}

func setIndexedBitOperands(resolved *z80parser.ResolvedInstruction) error {
	switch resolved.Instruction {
	case cpuz80.DdcbShift, cpuz80.FdcbShift:
		resolved.OperandValues = []ast.Node{ast.NewNumber(5)}
		return nil

	case cpuz80.DdcbBit, cpuz80.DdcbRes, cpuz80.DdcbSet,
		cpuz80.FdcbBit, cpuz80.FdcbRes, cpuz80.FdcbSet:
		resolved.OperandValues = []ast.Node{ast.NewNumber(3), ast.NewNumber(5)}
		return nil

	default:
		return fmt.Errorf("unsupported indexed bit instruction %q", resolved.Instruction.Name)
	}
}

func setOptionalByteOperand(resolved *z80parser.ResolvedInstruction, opcodeInfo cpuz80.OpcodeInfo) error {
	operandWidth := int(opcodeInfo.Size) - baseOpcodeWidth(opcodeInfo)
	switch operandWidth {
	case 0:
		return nil
	case 1:
		resolved.OperandValues = []ast.Node{ast.NewNumber(0x20)}
		return nil
	default:
		return fmt.Errorf("unsupported optional-byte operand width %d for instruction %q", operandWidth, resolved.Instruction.Name)
	}
}

func baseOpcodeWidth(opcodeInfo cpuz80.OpcodeInfo) int {
	if opcodeInfo.Prefix != 0 {
		return 2
	}
	return 1
}
