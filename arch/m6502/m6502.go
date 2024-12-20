// Package m6502 provides a 6502 architecture specific assembler code.
package m6502

import (
	"github.com/retroenv/retroasm/arch"
	"github.com/retroenv/retroasm/arch/m6502/assembler"
	"github.com/retroenv/retroasm/arch/m6502/parser"
	"github.com/retroenv/retroasm/assembler/config"
	"github.com/retroenv/retroasm/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/m6502"
)

// New returns a new 6502 architecture configuration.
func New() *config.Config[*m6502.Instruction] {
	p := &arch6502[*m6502.Instruction]{}
	cfg := &config.Config[*m6502.Instruction]{
		Arch: p,
	}
	return cfg
}

type arch6502[T any] struct {
}

func (_ *arch6502[T]) AddressWidth() int {
	return 16
}

func (_ *arch6502[T]) Instruction(name string) (*m6502.Instruction, bool) {
	ins, ok := m6502.Instructions[name]
	return ins, ok
}

// nolint: wrapcheck
func (_ *arch6502[T]) ParseIdentifier(p arch.Parser, ins *m6502.Instruction) (ast.Node, error) {
	return parser.ParseIdentifier(p, ins)
}

// nolint: wrapcheck
func (_ *arch6502[T]) AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	return assembler.AssignInstructionAddress(assigner, ins)
}

// nolint: wrapcheck
func (_ *arch6502[T]) GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	return assembler.GenerateInstructionOpcode(assigner, ins)
}
