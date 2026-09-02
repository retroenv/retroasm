// Package cpu68000 provides Motorola 68000 architecture-specific assembler support.
package cpu68000

import (
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/cpu68000/assembler"
	"github.com/retroenv/retroasm/pkg/arch/cpu68000/parser"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	retroarch "github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/arch/cpu/cpu68000"
)

// New returns a new CPU68000 architecture configuration.
func New() *config.Config[*cpu68000.Instruction] {
	p := &architecture{}
	cfg := &config.Config[*cpu68000.Instruction]{
		Arch: p,
	}
	return cfg
}

type architecture struct{}

func (ar *architecture) AddressWidth() int {
	return 24
}

func (ar *architecture) BuildInstruction(
	mnemonic string,
	operands parser.Operands,
) (ast.Instruction, error) {

	instruction, ok := ar.Instruction(strings.TrimSpace(mnemonic))
	if !ok {
		return ast.Instruction{}, fmt.Errorf("unknown CPU68000 instruction %q", mnemonic)
	}
	built, err := parser.BuildInstruction(mnemonic, instruction, operands)
	if err != nil {
		return ast.Instruction{}, err //nolint:wrapcheck // architecture codec boundary adds context
	}
	built.SetOpcodeID(ar.OpcodeID(instruction))
	return built, nil
}

func (ar *architecture) Instruction(name string) (*cpu68000.Instruction, bool) {
	// Try exact match first (handles "Bcc", "Scc", "DBcc")
	if ins, ok := cpu68000.Instructions[strings.ToUpper(name)]; ok {
		return ins, ok
	}

	// Try condition code variants: BEQ -> Bcc, DBNE -> DBcc, SHI -> Scc
	baseName, _, hasCond := parser.ParseConditionCode(name)
	if hasCond {
		ins, ok := cpu68000.Instructions[baseName]
		return ins, ok
	}

	// Try stripping size suffix: MOVE.L -> MOVE
	base, _ := parser.ParseSizeSuffix(name)
	if base != name {
		upper := strings.ToUpper(base)
		if ins, ok := cpu68000.Instructions[upper]; ok {
			return ins, ok
		}
		// Also check condition code after stripping size
		baseName, _, hasCond = parser.ParseConditionCode(base)
		if hasCond {
			ins, ok := cpu68000.Instructions[baseName]
			return ins, ok
		}
	}

	return nil, false
}

func (ar *architecture) OpcodeID(ins *cpu68000.Instruction) ast.OpcodeID {
	return ast.NewOpcodeID(retroarch.CPU68000, uint16(cpu68000.NameToOpcodeID[ins.Name]))
}

func (ar *architecture) ParseIdentifier(p arch.Parser, mnemonic string, ins *cpu68000.Instruction) (ast.Node, error) {
	return parser.ParseIdentifier(p, ins, mnemonic) //nolint:wrapcheck // thin delegation to sub-package
}

func (ar *architecture) ValidateInstruction(instruction ast.Instruction) error {
	details, ok := ar.Instruction(instruction.Name)
	if !ok {
		return fmt.Errorf("unknown CPU68000 instruction %q", instruction.Name)
	}
	expectedID := ar.OpcodeID(details)
	if instruction.OpcodeID != expectedID {
		return fmt.Errorf(
			"CPU68000 opcode identity %+v does not match mnemonic %q identity %+v",
			instruction.OpcodeID,
			instruction.Name,
			expectedID,
		)
	}
	return parser.ValidateInstruction(instruction, details) //nolint:wrapcheck // architecture codec boundary adds context
}

func (ar *architecture) FormatInstruction(instruction ast.Instruction) (string, error) {
	if err := ar.ValidateInstruction(instruction); err != nil {
		return "", err
	}
	return parser.FormatInstruction(instruction) //nolint:wrapcheck // architecture codec boundary adds context
}

func (ar *architecture) AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	return assembler.AssignInstructionAddress(assigner, ins) //nolint:wrapcheck // thin delegation to sub-package
}

func (ar *architecture) GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	return assembler.GenerateInstructionOpcode(assigner, ins) //nolint:wrapcheck // thin delegation to sub-package
}
