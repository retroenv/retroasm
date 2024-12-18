package assembler

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/retroenv/retroasm/arch"
	"github.com/retroenv/retroasm/scope"
	"github.com/retroenv/retrogolib/arch/cpu/m6502"
)

// generateOpcodesStep generates the opcodes for instructions and data nodes and resolves any
// references to their value or assigned addresses.
func generateOpcodesStep(asm *Assembler) error {
	currentScope := asm.fileScope

	for _, seg := range asm.segmentsOrder {
		for _, node := range seg.nodes {
			switch n := node.(type) {
			case *data:
				if err := generateReferenceDataBytes(currentScope, n); err != nil {
					return fmt.Errorf("generating data node opcode: %w", err)
				}
				if n.fill {
					if err := generateDataFillBytes(n); err != nil {
						return fmt.Errorf("generating data node opcode: %w", err)
					}
				}

			case *instruction:
				if err := generateInstructionOpcode(currentScope, n, asm.cfg.Arch); err != nil {
					return fmt.Errorf("generating instruction node opcode: %w", err)
				}

			case scopeChange:
				currentScope = n.scope
			}
		}
	}
	return nil
}

// generateDataFillBytes fills a reserved buffer.
func generateDataFillBytes(d *data) error {
	size, err := d.size.IntValue()
	if err != nil {
		return fmt.Errorf("getting data node size: %w", err)
	}

	var filler []byte
	for _, val := range d.values {
		b, ok := val.([]byte)
		if !ok {
			return fmt.Errorf("unsupported node value type %T", val)
		}
		filler = append(filler, b...)
	}

	b := make([]byte, size)
	if len(filler) > 0 {
		j := 0
		for i := range b {
			if j >= len(filler) {
				j = 0
			}
			b[i] = filler[j]
			j++
		}
	}

	// replace the defined filler values with the final filled reserved buffer
	d.values = []any{b}
	return nil
}

// generateReferenceDataBytes generates bytes for the data node by resolving any data or address references.
func generateReferenceDataBytes(currentScope *scope.Scope, d *data) error {
	for i, item := range d.values {
		ref, ok := item.(reference)
		if !ok {
			continue
		}

		sym, err := currentScope.GetSymbol(ref.name)
		if err != nil {
			return fmt.Errorf("getting instruction argument: %w", err)
		}

		value, err := sym.Value(currentScope)
		if err != nil {
			return fmt.Errorf("getting symbol '%s' value: %w", ref.name, err)
		}

		var address uint64

		switch v := value.(type) {
		case int64:
			address = uint64(v)
		case uint64:
			address = v
		default:
			return fmt.Errorf("unexpected reference value type %T", value)
		}

		var b []byte

		switch ref.typ {
		case fullAddress:
			b = []byte{byte(address), byte(address >> 8)}
		case lowAddressByte:
			b = []byte{byte(address)}
		case highAddressByte:
			b = []byte{byte(address >> 8)}
		default:
			return fmt.Errorf("unsupported reference type %d", ref.typ)
		}

		d.values[i] = b
	}
	return nil
}

// generateInstructionOpcode generates the instruction opcode based on the instruction base opcode,
// its addressing mode and parameters.
func generateInstructionOpcode(currentScope *scope.Scope, ins *instruction,
	arch arch.Architecture) error {

	instructionInfo := arch.Instructions[ins.name]
	addressingInfo := instructionInfo.Addressing[ins.addressing]
	ins.opcodes = []byte{addressingInfo.Opcode}
	ins.size = int(addressingInfo.Size)

	switch ins.addressing {
	case m6502.ImpliedAddressing, m6502.AccumulatorAddressing:

	case m6502.ImmediateAddressing:
		if err := generateImmediateAddressingOpcode(ins, currentScope); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case m6502.AbsoluteAddressing, m6502.AbsoluteXAddressing, m6502.AbsoluteYAddressing,
		m6502.IndirectAddressing, m6502.IndirectXAddressing, m6502.IndirectYAddressing:
		if err := generateAbsoluteIndirectAddressingOpcode(ins, currentScope); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case m6502.ZeroPageAddressing, m6502.ZeroPageXAddressing, m6502.ZeroPageYAddressing:
		if err := generateZeroPageAddressingOpcode(ins, currentScope); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	case m6502.RelativeAddressing:
		if err := generateRelativeAddressingOpcode(ins, currentScope); err != nil {
			return fmt.Errorf("generating opcode: %w", err)
		}

	default:
		return fmt.Errorf("unsupported instruction addressing %d", ins.addressing)
	}

	return nil
}

func generateAbsoluteIndirectAddressingOpcode(ins *instruction, currentScope *scope.Scope) error {
	value, err := getArgumentValue(ins.argument, currentScope)
	if err != nil {
		return fmt.Errorf("getting instruction argument: %w", err)
	}
	if value > math.MaxUint16 {
		return fmt.Errorf("value %d exceeds word", value)
	}

	ins.opcodes = binary.LittleEndian.AppendUint16(ins.opcodes, uint16(value))
	return nil
}

func generateZeroPageAddressingOpcode(ins *instruction, currentScope *scope.Scope) error {
	value, err := getArgumentValue(ins.argument, currentScope)
	if err != nil {
		return fmt.Errorf("getting instruction argument: %w", err)
	}
	if value > math.MaxUint8 {
		return fmt.Errorf("value %d exceeds byte", value)
	}

	ins.opcodes = append(ins.opcodes, byte(value))
	return nil
}

func generateImmediateAddressingOpcode(ins *instruction, currentScope *scope.Scope) error {
	value, err := getArgumentValue(ins.argument, currentScope)
	if err != nil {
		return fmt.Errorf("getting instruction argument: %w", err)
	}
	if value > math.MaxUint8 {
		return fmt.Errorf("value %d exceeds byte", value)
	}

	ins.opcodes = append(ins.opcodes, byte(value))
	return nil
}

func generateRelativeAddressingOpcode(ins *instruction, currentScope *scope.Scope) error {
	value, err := getArgumentValue(ins.argument, currentScope)
	if err != nil {
		return fmt.Errorf("getting instruction argument: %w", err)
	}

	b, err := getRelativeOffset(value, ins.address+uint64(ins.size))
	if err != nil {
		return fmt.Errorf("value %d exceeds byte", value)
	}

	ins.opcodes = append(ins.opcodes, b)
	return nil
}

func getArgumentValue(argument any, currentScope *scope.Scope) (uint64, error) {
	switch arg := argument.(type) {
	case uint64:
		return arg, nil

	case reference:
		sym, err := currentScope.GetSymbol(arg.name)
		if err != nil {
			return 0, fmt.Errorf("getting instruction argument: %w", err)
		}

		value, err := sym.Value(currentScope)
		if err != nil {
			return 0, fmt.Errorf("getting symbol '%s' value: %w", arg.name, err)
		}

		switch v := value.(type) {
		case int64:
			return uint64(v), nil
		case uint64:
			return v, nil
		default:
			return 0, fmt.Errorf("unexpected argument value type %T", value)
		}

	default:
		return 0, fmt.Errorf("unexpected argument type %T", arg)
	}
}

func getRelativeOffset(destination, addressAfterInstruction uint64) (byte, error) {
	diff := int64(destination) - int64(addressAfterInstruction)

	switch {
	case diff < -128 || diff > 127:
		return 0, fmt.Errorf("relative distance %d exceeds limit", diff)

	case diff >= 0:
		return byte(diff), nil

	default:
		return byte(256 + diff), nil
	}
}
