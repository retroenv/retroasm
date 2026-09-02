// Package chip8 provides a Chip-8 architecture specific assembler code.
package chip8

import (
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/chip8/assembler"
	"github.com/retroenv/retroasm/pkg/arch/chip8/parser"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	retroarch "github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/arch/cpu/chip8"
)

// New returns a new Chip-8 architecture configuration.
func New() *config.Config[*chip8.Instruction] {
	p := &archChip8[*chip8.Instruction]{}
	segment := &config.Segment{
		Memory: config.Memory{
			Name:  "CHIP8",
			Start: 0x200,
			Size:  0xE00,
		},
		SegmentName:  "CODE",
		SegmentStart: 0x200,
	}
	cfg := &config.Config[*chip8.Instruction]{
		Arch:     p,
		Segments: map[string]*config.Segment{segment.SegmentName: segment},
		SegmentsOrdered: []*config.Segment{
			segment,
		},
	}
	return cfg
}

type archChip8[T any] struct {
}

func (_ *archChip8[T]) AddressWidth() int {
	return 12
}

func (ar *archChip8[T]) BuildInstruction(
	mnemonic string,
	operands parser.Operands,
) (ast.Instruction, error) {

	lookupName := strings.ToLower(strings.TrimSpace(mnemonic))
	instruction, ok := ar.Instruction(lookupName)
	if !ok {
		return ast.Instruction{}, fmt.Errorf("unknown CHIP-8 instruction %q", mnemonic)
	}
	built, err := parser.BuildInstruction(lookupName, instruction, operands)
	if err != nil {
		return ast.Instruction{}, err //nolint:wrapcheck // architecture codec boundary adds context
	}
	built.SetOpcodeID(ar.OpcodeID(instruction))
	return built, nil
}

func (_ *archChip8[T]) Instruction(name string) (*chip8.Instruction, bool) {
	ins, ok := chip8.Instructions[name]
	return ins, ok
}

func (_ *archChip8[T]) OpcodeID(ins *chip8.Instruction) ast.OpcodeID {
	return ast.NewOpcodeID(retroarch.CHIP8, uint16(chip8.NameToOpcodeID[ins.Name]))
}

func (ar *archChip8[T]) ValidateInstruction(instruction ast.Instruction) error {
	details, ok := ar.Instruction(strings.ToLower(strings.TrimSpace(instruction.Name)))
	if !ok {
		return fmt.Errorf("unknown CHIP-8 instruction %q", instruction.Name)
	}
	expectedID := ar.OpcodeID(details)
	if instruction.OpcodeID != expectedID {
		return fmt.Errorf(
			"CHIP-8 opcode identity %+v does not match mnemonic %q identity %+v",
			instruction.OpcodeID,
			instruction.Name,
			expectedID,
		)
	}
	return parser.ValidateInstruction(instruction, details) //nolint:wrapcheck // architecture codec boundary adds context
}

func (ar *archChip8[T]) FormatInstruction(instruction ast.Instruction) (string, error) {
	if err := ar.ValidateInstruction(instruction); err != nil {
		return "", err
	}
	return parser.FormatInstruction(instruction) //nolint:wrapcheck // architecture codec boundary adds context
}

// nolint: wrapcheck
func (_ *archChip8[T]) ParseIdentifier(p arch.Parser, _ string, ins *chip8.Instruction) (ast.Node, error) {
	return parser.ParseIdentifier(p, ins)
}

// nolint: wrapcheck
func (_ *archChip8[T]) AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	return assembler.AssignInstructionAddress(assigner, ins)
}

// nolint: wrapcheck
func (_ *archChip8[T]) GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	return assembler.GenerateInstructionOpcode(assigner, ins)
}
