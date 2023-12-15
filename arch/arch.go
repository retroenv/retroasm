// Package arch contains types and functions used for multi architecture support.
package arch

import (
	"github.com/retroenv/retrogolib/arch/cpu/m6502"
	"github.com/retroenv/retrogolib/cpu"
)

// Architecture contains architecture specific information.
type Architecture struct {
	// AddressWidth describes the memory address widths of the CPU.
	AddressWidth int

	// BranchingInstructions contains all CPU branching instruction names.
	BranchingInstructions map[string]struct{}

	// Instructions maps instruction names to CPU instruction information.
	Instructions map[string]*cpu.Instruction
}

// NewNES returns a new NES architecture instance.
func NewNES() Architecture {
	return Architecture{
		AddressWidth:          16,
		BranchingInstructions: m6502.BranchingInstructions,
		Instructions:          m6502.Instructions,
	}
}
