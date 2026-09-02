package parser

import (
	"errors"
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu65816"
)

const (
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

// State is the M/X width state before one CPU65816 instruction.
type State struct {
	AccumulatorWidth Width
	IndexWidth       Width
}

// DefaultState returns the compiler and assembler entry state with 8-bit M/X widths.
func DefaultState() State {
	return State{AccumulatorWidth: WidthByte, IndexWidth: WidthByte}
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
	return nil
}

func validWidth(width Width) bool {
	return width == WidthUnknown || width == WidthByte || width == WidthWord
}

func nextState(state State, instruction *cpu65816.Instruction, operands Operands) State {
	switch instruction.Name {
	case cpu65816.PlpName, cpu65816.RtiName, cpu65816.XceName:
		// These instructions restore or derive M/X from runtime processor state.
		return State{}
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

	width := WidthWord
	if mnemonic == cpu65816.SepName {
		width = WidthByte
	}
	if mask&statusAccumulatorWidth != 0 {
		state.AccumulatorWidth = width
	}
	if mask&statusIndexWidth != 0 {
		state.IndexWidth = width
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
