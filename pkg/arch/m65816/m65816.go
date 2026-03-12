// Package m65816 provides a WDC 65C816 architecture specific assembler code.
package m65816

import (
	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/m65816/assembler"
	"github.com/retroenv/retroasm/pkg/arch/m65816/parser"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/m65816"
)

// New returns a new 65816 architecture configuration.
func New() *config.Config[*m65816.Instruction] {
	p := &arch65816[*m65816.Instruction]{}
	cfg := &config.Config[*m65816.Instruction]{
		Arch: p,
	}
	return cfg
}

type arch65816[T any] struct {
}

func (ar *arch65816[T]) AddressWidth() int {
	return 24
}

func (ar *arch65816[T]) Instruction(name string) (*m65816.Instruction, bool) {
	ins, ok := m65816.Instructions[name]
	return ins, ok
}

func (ar *arch65816[T]) ParseIdentifier(p arch.Parser, ins *m65816.Instruction) (ast.Node, error) {
	return parser.ParseIdentifier(p, ins) //nolint:wrapcheck // thin delegation to sub-package
}

func (ar *arch65816[T]) AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	return assembler.AssignInstructionAddress(assigner, ins) //nolint:wrapcheck // thin delegation to sub-package
}

func (ar *arch65816[T]) GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	return assembler.GenerateInstructionOpcode(assigner, ins) //nolint:wrapcheck // thin delegation to sub-package
}
