package parser

import (
	"errors"
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu65816"
)

const (
	statusCarry            = 0x01
	statusIndexWidth       = 0x10
	statusAccumulatorWidth = 0x20
)

var errInvalidState = errors.New("invalid CPU65816 stream state")

// Width is a known CPU65816 register width in bytes. Zero means runtime-dependent.
type Width uint8

const (
	WidthUnknown Width = iota
	WidthByte
	WidthWord
)

// StatusBit is a known CPU65816 processor-status bit. Zero means runtime-dependent.
type StatusBit uint8

const (
	StatusUnknown StatusBit = iota
	StatusClear
	StatusSet
)

// State is the processor state needed to resolve width-sensitive instructions.
type State struct {
	AccumulatorWidth Width
	IndexWidth       Width
	Carry            StatusBit
	Emulation        StatusBit
}

// DefaultState returns the compiler and assembler entry state in native mode with 8-bit M/X widths.
func DefaultState() State {
	return State{
		AccumulatorWidth: WidthByte,
		IndexWidth:       WidthByte,
		Emulation:        StatusClear,
	}
}

func stateFromParser(p arch.Parser) (State, arch.StatefulParser, error) {
	stateful, ok := p.(arch.StatefulParser)
	if !ok {
		return DefaultState(), nil, nil
	}
	raw := stateful.ArchitectureState()
	if raw == nil {
		state := DefaultState()
		stateful.SetArchitectureState(state)
		return state, stateful, nil
	}
	state, ok := raw.(State)
	if !ok {
		return State{}, stateful, fmt.Errorf("%w: got %T", errInvalidState, raw)
	}
	if err := state.validate(); err != nil {
		return State{}, stateful, err
	}
	return state, stateful, nil
}

func (state State) validate() error {
	if !validWidth(state.AccumulatorWidth) {
		return fmt.Errorf("%w: accumulator width %d", errInvalidState, state.AccumulatorWidth)
	}
	if !validWidth(state.IndexWidth) {
		return fmt.Errorf("%w: index width %d", errInvalidState, state.IndexWidth)
	}
	if !validStatusBit(state.Carry) {
		return fmt.Errorf("%w: carry state %d", errInvalidState, state.Carry)
	}
	if !validStatusBit(state.Emulation) {
		return fmt.Errorf("%w: emulation state %d", errInvalidState, state.Emulation)
	}
	if state.Emulation == StatusSet &&
		(state.AccumulatorWidth != WidthByte || state.IndexWidth != WidthByte) {

		return fmt.Errorf("%w: emulation mode requires 8-bit M/X widths", errInvalidState)
	}
	return nil
}

func validWidth(width Width) bool {
	return width == WidthUnknown || width == WidthByte || width == WidthWord
}

func validStatusBit(bit StatusBit) bool {
	return bit == StatusUnknown || bit == StatusClear || bit == StatusSet
}

func nextState(state State, instruction *cpu65816.Instruction, operands Operands) State {
	switch instruction.Name {
	case cpu65816.ClcName:
		state.Carry = StatusClear
		return state
	case cpu65816.SecName:
		state.Carry = StatusSet
		return state
	case cpu65816.PlpName, cpu65816.RtiName:
		return stateAfterStatusRestore(state)
	case cpu65816.XceName:
		return stateAfterExchangeCarryEmulation(state)
	case cpu65816.RepName, cpu65816.SepName:
		return applyStatusMask(state, instruction.Name, operands)
	default:
		return state
	}
}

func applyStatusMask(state State, mnemonic string, operands Operands) State {
	if len(operands) != 1 {
		return State{}
	}
	mask, ok := ast.NumberValue(operands[0].Value)
	if !ok {
		// A runtime mask may change either width, so both become unknown.
		return State{}
	}

	if mask&statusCarry != 0 {
		state.Carry = StatusClear
		if mnemonic == cpu65816.SepName {
			state.Carry = StatusSet
		}
	}
	if mask&statusAccumulatorWidth != 0 {
		state.AccumulatorWidth = statusWidth(state, mnemonic)
	}
	if mask&statusIndexWidth != 0 {
		state.IndexWidth = statusWidth(state, mnemonic)
	}
	return state
}

func statusWidth(state State, mnemonic string) Width {
	if mnemonic == cpu65816.SepName || state.Emulation == StatusSet {
		return WidthByte
	}
	if state.Emulation == StatusClear {
		return WidthWord
	}
	return WidthUnknown
}

func stateAfterStatusRestore(state State) State {
	state.Carry = StatusUnknown
	if state.Emulation == StatusSet {
		state.AccumulatorWidth = WidthByte
		state.IndexWidth = WidthByte
		return state
	}
	state.AccumulatorWidth = WidthUnknown
	state.IndexWidth = WidthUnknown
	return state
}

func stateAfterExchangeCarryEmulation(state State) State {
	state.Carry, state.Emulation = state.Emulation, state.Carry
	switch state.Emulation {
	case StatusSet:
		state.AccumulatorWidth = WidthByte
		state.IndexWidth = WidthByte
	case StatusUnknown:
		// Both possible XCE outcomes retain byte widths when M/X are already set.
		if state.AccumulatorWidth != WidthByte || state.IndexWidth != WidthByte {
			state.AccumulatorWidth = WidthUnknown
			state.IndexWidth = WidthUnknown
		}
	}
	return state
}

func immediateWidth(instruction *cpu65816.Instruction, state State) Width {
	switch instruction.Name {
	case cpu65816.AdcName, cpu65816.AndName, cpu65816.BitName, cpu65816.CmpName,
		cpu65816.EorName, cpu65816.LdaName, cpu65816.OraName, cpu65816.SbcName:
		return state.AccumulatorWidth
	case cpu65816.CpxName, cpu65816.CpyName, cpu65816.LdxName, cpu65816.LdyName:
		return state.IndexWidth
	default:
		return WidthByte
	}
}
