// Package cpu68000 provides Motorola 68000 architecture-specific assembler support.
package cpu68000

import (
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/cpu68000/assembler"
	"github.com/retroenv/retroasm/pkg/arch/cpu68000/parser"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/parser/ast"
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

type architecture struct {
	lastMnemonic string // original mnemonic from Instruction() lookup, used by ParseIdentifier()
}

func (ar *architecture) AddressWidth() int {
	return 24
}

func (ar *architecture) Instruction(name string) (*cpu68000.Instruction, bool) {
	ar.lastMnemonic = name

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

func (ar *architecture) ParseIdentifier(p arch.Parser, ins *cpu68000.Instruction) (ast.Node, error) {
	return parser.ParseIdentifier(p, ins, ar.lastMnemonic) //nolint:wrapcheck // thin delegation to sub-package
}

func (ar *architecture) AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	return assembler.AssignInstructionAddress(assigner, ins) //nolint:wrapcheck // thin delegation to sub-package
}

func (ar *architecture) GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	return assembler.GenerateInstructionOpcode(assigner, ins) //nolint:wrapcheck // thin delegation to sub-package
}
