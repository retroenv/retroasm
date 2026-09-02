// Package assembler implements the architecture specific assembler functionality.
package assembler

import (
	"errors"
	"fmt"
	"math"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/cpu65816/parser"
	"github.com/retroenv/retroasm/pkg/number"
	"github.com/retroenv/retrogolib/arch/cpu/cpu65816"
)

var errUnsupportedArgumentType = errors.New("unsupported CPU65816 argument type")

// AssignInstructionAddress assigns an address to the instruction and returns the next program counter.
func AssignInstructionAddress(assigner arch.AddressAssigner, ins arch.Instruction) (uint64, error) {
	pc := assigner.ProgramCounter()
	ins.SetAddress(pc)

	resolved, err := resolvedInstruction(ins.Argument())
	if err != nil {
		return 0, fmt.Errorf("resolving instruction argument: %w", err)
	}

	addressing, err := resolveAddressingMode(assigner, ins, resolved)
	if err != nil {
		return 0, err
	}
	size, err := resolved.EncodedSize(addressing)
	if err != nil {
		return 0, fmt.Errorf("resolving instruction size: %w", err)
	}
	ins.SetSize(size)

	return pc + uint64(size), nil
}

// disambiguousAddressing maps ambiguous addressing modes to their absolute and
// direct page variants. The assembler resolves these during address assignment
// based on whether the argument value fits in a byte.
var disambiguousAddressing = map[cpu65816.AddressingMode][2]cpu65816.AddressingMode{
	parser.AbsoluteDirectPageAddressing: {cpu65816.AbsoluteAddressing, cpu65816.DirectPageAddressing},
	parser.XAddressing:                  {cpu65816.AbsoluteIndexedXAddressing, cpu65816.DirectPageIndexedXAddressing},
	parser.YAddressing:                  {cpu65816.AbsoluteIndexedYAddressing, cpu65816.DirectPageIndexedYAddressing},
}

func resolvedInstruction(argument any) (parser.ResolvedInstruction, error) {
	resolved, ok := argument.(parser.ResolvedInstruction)
	if !ok {
		return parser.ResolvedInstruction{}, fmt.Errorf("%w: %T", errUnsupportedArgumentType, argument)
	}
	return resolved, nil
}

func resolveAddressingMode(
	assigner arch.AddressAssigner,
	ins arch.Instruction,
	resolved parser.ResolvedInstruction,
) (cpu65816.AddressingMode, error) {

	addressing := resolved.Addressing
	modes, ok := disambiguousAddressing[addressing]
	if !ok {
		ins.SetAddressing(int(addressing))
		return addressing, nil
	}

	value, err := resolvedOperandValue(assigner, resolved, 0)
	if err != nil {
		return cpu65816.NoAddressing, fmt.Errorf("getting instruction argument: %w", err)
	}

	if value > math.MaxUint8 {
		addressing = modes[0]
	} else {
		addressing = modes[1]
	}
	ins.SetAddressing(int(addressing))
	return addressing, nil
}

func resolvedOperandValue(
	assigner arch.AddressAssigner,
	resolved parser.ResolvedInstruction,
	index int,
) (uint64, error) {

	if index < 0 || index >= len(resolved.Operands) || resolved.Operands[index].Value == nil {
		return 0, fmt.Errorf("operand %d is missing", index)
	}
	value, err := assigner.ArgumentValue(resolved.Operands[index].Value)
	if err != nil {
		return 0, err //nolint:wrapcheck // caller supplies instruction context
	}
	for _, modifier := range resolved.Operands[index].Modifiers {
		offset, err := number.Parse(modifier.Value)
		if err != nil {
			return 0, fmt.Errorf("parsing modifier %q: %w", modifier.Value, err)
		}
		switch modifier.Operator.Operator {
		case "+":
			if math.MaxUint64-value < offset {
				return 0, errors.New("instruction argument modifier overflows uint64")
			}
			value += offset
		case "-":
			if offset > value {
				return 0, errors.New("instruction argument modifier produces a negative value")
			}
			value -= offset
		default:
			return 0, fmt.Errorf("unsupported modifier operator %q", modifier.Operator.Operator)
		}
	}
	return value, nil
}
