// Package sm83 provides SM83 (Game Boy CPU) architecture-specific assembler support.
package sm83

import (
	"fmt"
	"slices"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	sm83assembler "github.com/retroenv/retroasm/pkg/arch/sm83/assembler"
	sm83parser "github.com/retroenv/retroasm/pkg/arch/sm83/parser"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	retroarch "github.com/retroenv/retrogolib/arch"
	cpusm83 "github.com/retroenv/retrogolib/arch/cpu/sm83"
)

// InstructionGroup contains all instruction variants for a mnemonic.
type InstructionGroup struct {
	Name     string
	Variants []*cpusm83.Instruction
}

// New returns a new SM83 architecture configuration.
func New() *config.Config[*InstructionGroup] {
	p := newArchitecture()
	cfg := &config.Config[*InstructionGroup]{
		Arch: p,
	}
	return cfg
}

type architecture struct {
	instructionGroups map[string]*InstructionGroup
}

func newArchitecture() *architecture {
	return &architecture{
		instructionGroups: buildInstructionGroups(),
	}
}

func (ar *architecture) AddressWidth() int {
	return 16
}

func (*architecture) ByteOrder() ast.ByteOrder {
	return ast.ByteOrderLittle
}

func (ar *architecture) AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	return sm83assembler.AssignInstructionAddress(assigner, ins) //nolint:wrapcheck // thin delegation to sub-package
}

func (ar *architecture) GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	return sm83assembler.GenerateInstructionOpcode(assigner, ins) //nolint:wrapcheck // thin delegation to sub-package
}

func (ar *architecture) Instruction(name string) (*InstructionGroup, bool) {
	group, ok := ar.instructionGroups[strings.ToLower(name)]
	return group, ok
}

func (ar *architecture) OpcodeID(ins *InstructionGroup) ast.OpcodeID {
	return ast.NewOpcodeID(retroarch.SM83, uint16(cpusm83.NameToOpcodeID[ins.Name]))
}

func (ar *architecture) ParseIdentifier(p arch.Parser, _ string, ins *InstructionGroup) (ast.Node, error) {
	return sm83parser.ParseIdentifier(p, ins.Name, ins.Variants) //nolint:wrapcheck // thin delegation to sub-package
}

func (ar *architecture) BuildInstruction(
	mnemonic string,
	operands sm83parser.Operands,
) (ast.Instruction, error) {

	group, ok := ar.Instruction(mnemonic)
	if !ok {
		return ast.Instruction{}, fmt.Errorf("unknown SM83 instruction %q", mnemonic)
	}
	instruction, err := sm83parser.BuildInstruction(group.Name, group.Variants, operands...)
	if err != nil {
		return ast.Instruction{}, err //nolint:wrapcheck // architecture codec boundary adds context
	}
	instruction.SetOpcodeID(ar.OpcodeID(group))
	return instruction, nil
}

func (ar *architecture) ValidateInstruction(instruction ast.Instruction) error {
	group, ok := ar.Instruction(instruction.Name)
	if !ok {
		return fmt.Errorf("unknown SM83 instruction %q", instruction.Name)
	}
	expectedID := ar.OpcodeID(group)
	if instruction.OpcodeID != expectedID {
		return fmt.Errorf(
			"SM83 opcode identity %+v does not match mnemonic %q identity %+v",
			instruction.OpcodeID,
			instruction.Name,
			expectedID,
		)
	}
	return sm83parser.ValidateInstruction(instruction, group.Variants) //nolint:wrapcheck // architecture codec boundary adds context
}

func (ar *architecture) FormatInstruction(instruction ast.Instruction) (string, error) {
	if err := ar.ValidateInstruction(instruction); err != nil {
		return "", err
	}
	return sm83parser.FormatInstruction(instruction) //nolint:wrapcheck // architecture codec boundary adds context
}

func buildInstructionGroups() map[string]*InstructionGroup {
	instructionGroups := make(map[string]*InstructionGroup)

	addInstructionsFromOpcodeTable(instructionGroups, cpusm83.Opcodes)
	addInstructionSlice(instructionGroups, cbFamilyInstructions)

	return instructionGroups
}

var cbFamilyInstructions = []*cpusm83.Instruction{
	cpusm83.CBRlc,
	cpusm83.CBRrc,
	cpusm83.CBRl,
	cpusm83.CBRr,
	cpusm83.CBSla,
	cpusm83.CBSra,
	cpusm83.CBSwap,
	cpusm83.CBSrl,
	cpusm83.CBBit,
	cpusm83.CBRes,
	cpusm83.CBSet,
}

func addInstructionsFromOpcodeTable(instructionGroups map[string]*InstructionGroup, opcodes [256]cpusm83.Opcode) {
	for _, opcode := range opcodes {
		if opcode.Instruction == nil {
			continue
		}
		addInstruction(instructionGroups, opcode.Instruction)
	}
}

func addInstructionSlice(instructionGroups map[string]*InstructionGroup, instructions []*cpusm83.Instruction) {
	for _, ins := range instructions {
		if ins == nil {
			continue
		}
		addInstruction(instructionGroups, ins)
	}
}

func addInstruction(instructionGroups map[string]*InstructionGroup, ins *cpusm83.Instruction) {
	name := strings.ToLower(ins.Name)
	if name == "" {
		return
	}

	group, ok := instructionGroups[name]
	if !ok {
		group = &InstructionGroup{
			Name:     name,
			Variants: make([]*cpusm83.Instruction, 0, 2),
		}
		instructionGroups[name] = group
	}

	if containsInstruction(group.Variants, ins) {
		return
	}

	group.Variants = append(group.Variants, ins)
}

func containsInstruction(instructions []*cpusm83.Instruction, target *cpusm83.Instruction) bool {
	return slices.Contains(instructions, target)
}
