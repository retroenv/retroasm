// Package x86 provides a x86 (8086/286) architecture specific assembler code.
package x86

import (
	"fmt"

	"github.com/retroenv/retroasm/arch"
	"github.com/retroenv/retroasm/arch/x86/assembler"
	"github.com/retroenv/retroasm/arch/x86/parser"
	"github.com/retroenv/retroasm/assembler/config"
	"github.com/retroenv/retroasm/parser/ast"
)

// New returns a new x86 architecture configuration.
func New() *config.Config[*Instruction] {
	p := &archX86[*Instruction]{}
	cfg := &config.Config[*Instruction]{
		Arch: p,
	}
	return cfg
}

type archX86[T any] struct {
}

func (_ *archX86[T]) AddressWidth() int {
	return 16
}

func (_ *archX86[T]) Instruction(name string) (*Instruction, bool) {
	ins, ok := Instructions[name]
	return ins, ok
}

func (_ *archX86[T]) ParseIdentifier(p arch.Parser, ins *Instruction) (ast.Node, error) {
	node, err := parser.ParseIdentifier(p, ins)
	if err != nil {
		return nil, fmt.Errorf("parsing x86 instruction: %w", err)
	}
	return node, nil
}

func (_ *archX86[T]) AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	addr, err := assembler.AssignInstructionAddress(assigner, ins)
	if err != nil {
		return 0, fmt.Errorf("assigning x86 instruction address: %w", err)
	}
	return addr, nil
}

func (_ *archX86[T]) GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	err := assembler.GenerateInstructionOpcode(assigner, ins)
	if err != nil {
		return fmt.Errorf("generating x86 instruction opcode: %w", err)
	}
	return nil
}
