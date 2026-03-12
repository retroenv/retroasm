// Package assembler implements the architecture specific assembler functionality.
package assembler

import (
	"fmt"
	"math"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/m65816/parser"
	"github.com/retroenv/retrogolib/arch/cpu/m65816"
)

// AssignInstructionAddress assigns an address to the instruction and returns the next program counter.
func AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	pc := assigner.ProgramCounter()
	ins.SetAddress(pc)

	name := strings.ToLower(ins.Name())
	insDetails, ok := m65816.Instructions[name]
	if !ok {
		return 0, fmt.Errorf("unsupported instruction '%s'", name)
	}

	addressing := m65816.AddressingMode(ins.Addressing())

	if err := resolveAddressingMode(assigner, ins, addressing); err != nil {
		return 0, err
	}

	addressing = m65816.AddressingMode(ins.Addressing())
	addressingInfo, ok := insDetails.Addressing[addressing]
	if !ok {
		return 0, fmt.Errorf("unsupported instruction '%s' addressing %d", name, addressing)
	}

	programCounter := pc + uint64(addressingInfo.BaseSize)
	return programCounter, nil
}

// disambiguousAddressing maps ambiguous addressing modes to their absolute and
// direct page variants. The assembler resolves these during address assignment
// based on whether the argument value fits in a byte.
var disambiguousAddressing = map[m65816.AddressingMode][2]m65816.AddressingMode{
	parser.AbsoluteDirectPageAddressing: {m65816.AbsoluteAddressing, m65816.DirectPageAddressing},
	parser.XAddressing:                  {m65816.AbsoluteIndexedXAddressing, m65816.DirectPageIndexedXAddressing},
	parser.YAddressing:                  {m65816.AbsoluteIndexedYAddressing, m65816.DirectPageIndexedYAddressing},
}

func resolveAddressingMode(assigner arch.AddressAssigner, ins arch.Instruction, addressing m65816.AddressingMode) error {
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
