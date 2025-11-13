// Package chip8 provides a Chip-8 architecture specific assembler code.
package chip8

import (
	"github.com/retroenv/retroasm/arch"
	"github.com/retroenv/retroasm/arch/chip8/assembler"
	"github.com/retroenv/retroasm/arch/chip8/parser"
	"github.com/retroenv/retroasm/assembler/config"
	"github.com/retroenv/retroasm/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/chip8"
)

// New returns a new Chip-8 architecture configuration.
func New() *config.Config[*chip8.Instruction] {
	p := &archChip8[*chip8.Instruction]{}
	cfg := &config.Config[*chip8.Instruction]{
		Arch: p,
	}
	return cfg
}

type archChip8[T any] struct {
}

func (_ *archChip8[T]) AddressWidth() int {
	return 12
}

func (_ *archChip8[T]) Instruction(name string) (*chip8.Instruction, bool) {
	ins, ok := chip8.Instructions[name]
	return ins, ok
}

// nolint: wrapcheck
func (_ *archChip8[T]) ParseIdentifier(p arch.Parser, ins *chip8.Instruction) (ast.Node, error) {
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
