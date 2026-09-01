// Package cpu6502 provides a 6502 architecture specific assembler code.
package cpu6502

import (
	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/cpu6502/assembler"
	"github.com/retroenv/retroasm/pkg/arch/cpu6502/parser"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	retroarch "github.com/retroenv/retrogolib/arch"
	"github.com/retroenv/retrogolib/arch/cpu/cpu6502"
)

// New returns a new 6502 architecture configuration.
func New() *config.Config[*cpu6502.Instruction] {
	p := &arch6502[*cpu6502.Instruction]{}
	cfg := &config.Config[*cpu6502.Instruction]{
		Arch: p,
	}
	return cfg
}

type arch6502[T any] struct {
}

func (ar *arch6502[T]) AddressWidth() int {
	return 16
}

func (ar *arch6502[T]) Instruction(name string) (*cpu6502.Instruction, bool) {
	ins, ok := cpu6502.Instructions[name]
	return ins, ok
}

func (ar *arch6502[T]) OpcodeID(ins *cpu6502.Instruction) ast.OpcodeID {
	return ast.NewOpcodeID(retroarch.CPU6502, uint16(cpu6502.NameToOpcodeID[ins.Name]))
}

func (ar *arch6502[T]) ParseIdentifier(p arch.Parser, _ string, ins *cpu6502.Instruction) (ast.Node, error) {
	return parser.ParseIdentifier(p, ins) //nolint:wrapcheck // thin delegation to sub-package
}

func (ar *arch6502[T]) AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	return assembler.AssignInstructionAddress(assigner, ins) //nolint:wrapcheck // thin delegation to sub-package
}

func (ar *arch6502[T]) GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	return assembler.GenerateInstructionOpcode(assigner, ins) //nolint:wrapcheck // thin delegation to sub-package
}
