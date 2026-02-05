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

	// handle disambiguous addressing mode to reduce absolute addressings to
	// zeropage ones if the used address value fits into byte
	switch addressing {
	case parser.AbsoluteZeroPageAddressing:
		argument := ins.Argument()
		value, err := assigner.ArgumentValue(argument)
		if err != nil {
			return 0, fmt.Errorf("getting instruction argument: %w", err)
		}
		if value > math.MaxUint8 {
			ins.SetAddressing(int(m6502.AbsoluteAddressing))
		} else {
			ins.SetAddressing(int(m6502.ZeroPageAddressing))
		}

	case parser.XAddressing:
		argument := ins.Argument()
		value, err := assigner.ArgumentValue(argument)
		if err != nil {
			return 0, fmt.Errorf("getting instruction argument: %w", err)
		}
		if value > math.MaxUint8 {
			ins.SetAddressing(int(m6502.AbsoluteXAddressing))
		} else {
			ins.SetAddressing(int(m6502.ZeroPageXAddressing))
		}

	case parser.YAddressing:
		argument := ins.Argument()
		value, err := assigner.ArgumentValue(argument)
		if err != nil {
			return 0, fmt.Errorf("getting instruction argument: %w", err)
		}
		if value > math.MaxUint8 {
			ins.SetAddressing(int(m6502.AbsoluteYAddressing))
		} else {
			ins.SetAddressing(int(m6502.ZeroPageYAddressing))
		}
	}

	addressing = m6502.AddressingMode(ins.Addressing())
	addressingInfo, ok := insDetails.Addressing[addressing]
	if !ok {
		return 0, fmt.Errorf("unsupported instruction '%s' addressing %d", name, addressing)
	}

	programCounter := pc + uint64(addressingInfo.Size)
	return programCounter, nil
}
