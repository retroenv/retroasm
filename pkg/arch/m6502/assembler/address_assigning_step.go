// Package assembler implements the architecture specific assembler functionality.
package assembler

import (
	"fmt"
	"math"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/m6502/parser"
	"github.com/retroenv/retrogolib/arch/cpu/m6502"
)

func AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	pc := assigner.ProgramCounter()
	ins.SetAddress(pc)

	name := strings.ToLower(ins.Name())
	insDetails, ok := m6502.Instructions[name]
	if !ok {
		return 0, fmt.Errorf("unsupported instruction '%s'", name)
	}

	addressing := m6502.AddressingMode(ins.Addressing())

	// Resolve disambiguous addressing modes by checking whether the argument
	// value fits in a byte (zero page) or requires a word (absolute).
	if err := resolveAddressingMode(assigner, ins, addressing); err != nil {
		return 0, err
	}

	addressing = m6502.AddressingMode(ins.Addressing())
	addressingInfo, ok := insDetails.Addressing[addressing]
	if !ok {
		return 0, fmt.Errorf("unsupported instruction '%s' addressing %d", name, addressing)
	}

	programCounter := pc + uint64(addressingInfo.Size)
	return programCounter, nil
}

// disambiguousAddressing maps ambiguous addressing modes to their absolute and
// zero page variants. The assembler resolves these during address assignment
// based on whether the argument value fits in a byte.
var disambiguousAddressing = map[m6502.AddressingMode][2]m6502.AddressingMode{
	parser.AbsoluteZeroPageAddressing: {m6502.AbsoluteAddressing, m6502.ZeroPageAddressing},
	parser.XAddressing:                {m6502.AbsoluteXAddressing, m6502.ZeroPageXAddressing},
	parser.YAddressing:                {m6502.AbsoluteYAddressing, m6502.ZeroPageYAddressing},
}

func resolveAddressingMode(assigner arch.AddressAssigner, ins arch.Instruction, addressing m6502.AddressingMode) error {
	modes, ok := disambiguousAddressing[addressing]
	if !ok {
		return nil
	}

	value, err := assigner.ArgumentValue(ins.Argument())
	if err != nil {
		return fmt.Errorf("getting instruction argument: %w", err)
	}

	if value > math.MaxUint8 {
		ins.SetAddressing(int(modes[0]))
	} else {
		ins.SetAddressing(int(modes[1]))
	}
	return nil
}
