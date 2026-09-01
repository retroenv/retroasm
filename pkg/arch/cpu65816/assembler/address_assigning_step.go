// Package assembler implements the architecture specific assembler functionality.
package assembler

import (
	"fmt"
	"math"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/cpu65816/parser"
	"github.com/retroenv/retrogolib/arch/cpu/cpu65816"
)

// AssignInstructionAddress assigns an address to the instruction and returns the next program counter.
func AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	pc := assigner.ProgramCounter()
	ins.SetAddress(pc)

	name := strings.ToLower(ins.Name())
	insDetails, ok := cpu65816.Instructions[name]
	if !ok {
		return 0, fmt.Errorf("unsupported instruction '%s'", name)
	}

	addressing := cpu65816.AddressingMode(ins.Addressing())

	if err := resolveAddressingMode(assigner, ins, addressing); err != nil {
		return 0, err
	}

	addressing = cpu65816.AddressingMode(ins.Addressing())
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
var disambiguousAddressing = map[cpu65816.AddressingMode][2]cpu65816.AddressingMode{
	parser.AbsoluteDirectPageAddressing: {cpu65816.AbsoluteAddressing, cpu65816.DirectPageAddressing},
	parser.XAddressing:                  {cpu65816.AbsoluteIndexedXAddressing, cpu65816.DirectPageIndexedXAddressing},
	parser.YAddressing:                  {cpu65816.AbsoluteIndexedYAddressing, cpu65816.DirectPageIndexedYAddressing},
}

func resolveAddressingMode(assigner arch.AddressAssigner, ins arch.Instruction, addressing cpu65816.AddressingMode) error {
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
