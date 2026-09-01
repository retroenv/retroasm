// Package cpu65816 provides a WDC 65C816 architecture specific assembler code.
package cpu65816

import (
	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/cpu65816/assembler"
	"github.com/retroenv/retroasm/pkg/arch/cpu65816/parser"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	retroarch "github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/arch/cpu/cpu65816"
)

// New returns a new 65816 architecture configuration.
func New() *config.Config[*cpu65816.Instruction] {
	p := &arch65816[*cpu65816.Instruction]{}
	cfg := &config.Config[*cpu65816.Instruction]{
		Arch: p,
	}
	return cfg
}

type arch65816[T any] struct {
}

func (ar *arch65816[T]) AddressWidth() int {
	return 24
}

func (ar *arch65816[T]) Instruction(name string) (*cpu65816.Instruction, bool) {
	ins, ok := cpu65816.Instructions[name]
	return ins, ok
}

func (ar *arch65816[T]) OpcodeID(ins *cpu65816.Instruction) ast.OpcodeID {
	return ast.NewOpcodeID(retroarch.CPU65816, uint16(cpu65816.NameToOpcodeID[ins.Name]))
}

func (ar *arch65816[T]) ParseIdentifier(p arch.Parser, _ string, ins *cpu65816.Instruction) (ast.Node, error) {
	return parser.ParseIdentifier(p, ins) //nolint:wrapcheck // thin delegation to sub-package
}

func (ar *arch65816[T]) AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	return assembler.AssignInstructionAddress(assigner, ins) //nolint:wrapcheck // thin delegation to sub-package
}

func (ar *arch65816[T]) GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	return assembler.GenerateInstructionOpcode(assigner, ins) //nolint:wrapcheck // thin delegation to sub-package
}
