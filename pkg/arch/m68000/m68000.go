// Package m68000 provides Motorola 68000 architecture-specific assembler support.
package m68000

import (
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	m68000assembler "github.com/retroenv/retroasm/pkg/arch/m68000/assembler"
	m68000parser "github.com/retroenv/retroasm/pkg/arch/m68000/parser"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/m68000"
)

// New returns a new M68000 architecture configuration.
func New() *config.Config[*m68000.Instruction] {
	p := &architecture{}
	cfg := &config.Config[*m68000.Instruction]{
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

func (ar *architecture) Instruction(name string) (*m68000.Instruction, bool) {
	ar.lastMnemonic = name

	// Try exact match first (handles "Bcc", "Scc", "DBcc")
	if ins, ok := m68000.Instructions[strings.ToUpper(name)]; ok {
		return ins, ok
	}

	// Try condition code variants: BEQ -> Bcc, DBNE -> DBcc, SHI -> Scc
	baseName, _, hasCond := m68000parser.ParseConditionCode(name)
	if hasCond {
		ins, ok := m68000.Instructions[baseName]
		return ins, ok
	}

	// Try stripping size suffix: MOVE.L -> MOVE
	base, _ := m68000parser.ParseSizeSuffix(name)
	if base != name {
		upper := strings.ToUpper(base)
		if ins, ok := m68000.Instructions[upper]; ok {
			return ins, ok
		}
		// Also check condition code after stripping size
		baseName, _, hasCond = m68000parser.ParseConditionCode(base)
		if hasCond {
			ins, ok := m68000.Instructions[baseName]
			return ins, ok
		}
	}

	return nil, false
}

func (ar *architecture) ParseIdentifier(p arch.Parser, ins *m68000.Instruction) (ast.Node, error) {
	return m68000parser.ParseIdentifier(p, ins, ar.lastMnemonic) //nolint:wrapcheck // thin delegation to sub-package
}

func (ar *architecture) AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	return m68000assembler.AssignInstructionAddress(assigner, ins) //nolint:wrapcheck // thin delegation to sub-package
}

func (ar *architecture) GenerateInstructionOpcode(assigner arch.AddressAssigner, ins arch.Instruction) error {
	return m68000assembler.GenerateInstructionOpcode(assigner, ins) //nolint:wrapcheck // thin delegation to sub-package
}
