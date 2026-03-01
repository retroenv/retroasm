// Package z80 provides Z80 architecture-specific assembler support.
package z80

import (
	"slices"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	z80assembler "github.com/retroenv/retroasm/pkg/arch/z80/assembler"
	z80parser "github.com/retroenv/retroasm/pkg/arch/z80/parser"
	z80profile "github.com/retroenv/retroasm/pkg/arch/z80/profile"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
)

// InstructionGroup contains all instruction variants for a mnemonic.
type InstructionGroup struct {
	Name     string
	Variants []*cpuz80.Instruction
}

type architecture struct {
	instructionGroups map[string]*InstructionGroup
	profile           z80profile.Kind
}

// New returns a new Z80 architecture configuration.
func New(opts ...Option) *config.Config[*InstructionGroup] {
	settings := resolveOptions(opts)

	p := newArchitecture(settings)
	cfg := &config.Config[*InstructionGroup]{
		Arch: p,
	}
	return cfg
}

func newArchitecture(settings options) *architecture {
	return &architecture{
		instructionGroups: buildInstructionGroups(),
		profile:           settings.profile,
	}
}

func (ar *architecture) AddressWidth() int {
	return 16
}

func (ar *architecture) AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	return z80assembler.AssignInstructionAddress(assigner, ins) //nolint:wrapcheck // thin delegation to sub-package
}

func (ar *architecture) GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	return z80assembler.GenerateInstructionOpcode(assigner, ins) //nolint:wrapcheck // thin delegation to sub-package
}

func (ar *architecture) Instruction(name string) (*InstructionGroup, bool) {
	group, ok := ar.instructionGroups[strings.ToLower(name)]
	return group, ok
}

func (ar *architecture) ParseIdentifier(p arch.Parser, ins *InstructionGroup) (ast.Node, error) {
	return z80parser.ParseIdentifierWithProfile(p, ins.Name, ins.Variants, ar.profile) //nolint:wrapcheck // thin delegation to sub-package
}

func buildInstructionGroups() map[string]*InstructionGroup {
	instructionGroups := make(map[string]*InstructionGroup)

	addInstructionsFromOpcodeTable(instructionGroups, cpuz80.Opcodes)
	addInstructionsFromOpcodeTable(instructionGroups, cpuz80.EDOpcodes)
	addInstructionsFromOpcodeTable(instructionGroups, cpuz80.DDOpcodes)
	addInstructionsFromOpcodeTable(instructionGroups, cpuz80.FDOpcodes)
	addInstructionSlice(instructionGroups, cbFamilyInstructions)
	addInstructionSlice(instructionGroups, indexedBitInstructions)

	return instructionGroups
}

var cbFamilyInstructions = []*cpuz80.Instruction{
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
}

var indexedBitInstructions = []*cpuz80.Instruction{
	cpuz80.DdcbShift,
	cpuz80.DdcbBit,
	cpuz80.DdcbRes,
	cpuz80.DdcbSet,
	cpuz80.FdcbShift,
	cpuz80.FdcbBit,
	cpuz80.FdcbRes,
	cpuz80.FdcbSet,
}

func addInstructionsFromOpcodeTable(instructionGroups map[string]*InstructionGroup, opcodes [256]cpuz80.Opcode) {
	for _, opcode := range opcodes {
		if opcode.Instruction == nil {
			continue
		}
		addInstruction(instructionGroups, opcode.Instruction)
	}
}

func addInstructionSlice(instructionGroups map[string]*InstructionGroup, instructions []*cpuz80.Instruction) {
	for _, ins := range instructions {
		if ins == nil {
			continue
		}
		addInstruction(instructionGroups, ins)
	}
}

func addInstruction(instructionGroups map[string]*InstructionGroup, ins *cpuz80.Instruction) {
	name := strings.ToLower(ins.Name)
	if name == "" {
		return
	}

	group, ok := instructionGroups[name]
	if !ok {
		group = &InstructionGroup{
			Name:     name,
			Variants: make([]*cpuz80.Instruction, 0, 2),
		}
		instructionGroups[name] = group
	}

	if containsInstruction(group.Variants, ins) {
		return
	}

	group.Variants = append(group.Variants, ins)
}

func containsInstruction(instructions []*cpuz80.Instruction, target *cpuz80.Instruction) bool {
	return slices.Contains(instructions, target)
}
