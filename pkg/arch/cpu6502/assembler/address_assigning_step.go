// Package assembler implements the architecture specific assembler functionality.
package assembler

import (
	"errors"
	"fmt"
	"math"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/cpu6502/parser"
	"github.com/retroenv/retroasm/pkg/scope"
	"github.com/retroenv/retrogolib/arch/cpu/cpu6502"
)

func AssignInstructionAddress(
	assigner arch.AddressAssigner,
	ins arch.Instruction,
	insDetails *cpu6502.Instruction,
) (uint64, error) {

	pc := assigner.ProgramCounter()
	ins.SetAddress(pc)
	if insDetails == nil {
		return 0, fmt.Errorf("unsupported instruction '%s'", ins.Name())
	}

	addressing := cpu6502.AddressingMode(ins.Addressing())

	// Resolve disambiguous addressing modes by checking whether the argument
	// value fits in a byte (zero page) or requires a word (absolute).
	if err := resolveAddressingMode(assigner, ins, addressing); err != nil {
		return 0, err
	}

	addressing = cpu6502.AddressingMode(ins.Addressing())
	addressingInfo, ok := insDetails.Addressing[addressing]
	if !ok {
		return 0, fmt.Errorf("unsupported instruction '%s' addressing %d", ins.Name(), addressing)
	}

	programCounter := pc + uint64(addressingInfo.Size)
	return programCounter, nil
}

// disambiguousAddressing maps ambiguous addressing modes to their absolute and
// zero page variants. The assembler resolves these during address assignment
// based on whether the argument value fits in a byte.
var disambiguousAddressing = map[cpu6502.AddressingMode][2]cpu6502.AddressingMode{
	parser.AbsoluteZeroPageAddressing: {cpu6502.AbsoluteAddressing, cpu6502.ZeroPageAddressing},
	parser.XAddressing:                {cpu6502.AbsoluteXAddressing, cpu6502.ZeroPageXAddressing},
	parser.YAddressing:                {cpu6502.AbsoluteYAddressing, cpu6502.ZeroPageYAddressing},
}

func resolveAddressingMode(assigner arch.AddressAssigner, ins arch.Instruction, addressing cpu6502.AddressingMode) error {
	modes, ok := disambiguousAddressing[addressing]
	if !ok {
		return nil
	}

	value, err := assigner.ArgumentValue(ins.Argument())
	if errors.Is(err, scope.ErrForwardReference) {
		// The label's address hasn't been assigned yet (forward reference).
		// Conservatively use absolute (wider) addressing so the instruction
		// occupies the correct number of bytes; the actual target address is
		// filled in later by generateOpcodesStep once all labels are resolved.
		ins.SetAddressing(int(modes[0]))
		return nil
	}
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
