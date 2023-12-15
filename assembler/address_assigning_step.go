package assembler

import (
	"fmt"
	"math"

	"github.com/retroenv/assembler/parser/ast"
	"github.com/retroenv/assembler/scope"
	. "github.com/retroenv/retrogolib/addressing"
)

// assignAddressesStep assigns an address for every node in each scope.
func assignAddressesStep(asm *Assembler) error {
	var err error
	currentScope := asm.fileScope

	for _, seg := range asm.segmentsOrder {
		programCounter := seg.config.Start

		for _, node := range seg.nodes {
			switch n := node.(type) {
			case *data:
				programCounter, err = assignDataAddress(asm, n, programCounter)

			case *variable:
				assignVariableAddress(n, programCounter)

			case *scope.Symbol:
				err = assignSymbolAddress(asm, n, programCounter)

			case *instruction:
				programCounter, err = assignInstructionAddress(currentScope, asm, n, programCounter)

			case *base:
				programCounter, err = assignBaseAddress(n)

			case scopeChange:
				currentScope = n.scope

			case *ast.Configuration:

			default:
				return fmt.Errorf("unsupported node type %T", n)
			}

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func assignDataAddress(asm *Assembler, d *data, programCounter uint64) (uint64, error) {
	if d.size.IsEvaluatedAtAddressAssign() {
		_, err := d.size.EvaluateAtProgramCounter(asm.currentScope, d.width, programCounter)
		if err != nil {
			return 0, fmt.Errorf("evaluating data size at program counter: %w", err)
		}
	}

	d.address = programCounter
	size, err := d.size.IntValue()
	if err != nil {
		return 0, fmt.Errorf("getting data node size: %w", err)
	}
	programCounter += uint64(size)
	return programCounter, nil
}

func assignVariableAddress(v *variable, programCounter uint64) uint64 {
	v.address = programCounter
	programCounter += uint64(v.v.Size)
	return programCounter
}

func assignBaseAddress(b *base) (uint64, error) {
	i, err := b.address.IntValue()
	if err != nil {
		return 0, fmt.Errorf("getting base node address: %w", err)
	}
	return uint64(i), nil
}

func assignSymbolAddress(asm *Assembler, sym *scope.Symbol, programCounter uint64) error {
	sym.SetAddress(programCounter)
	exp := sym.Expression()
	if exp != nil && exp.IsEvaluatedAtAddressAssign() {
		_, err := exp.EvaluateAtProgramCounter(asm.currentScope, asm.cfg.Arch.AddressWidth, programCounter)
		if err != nil {
			return fmt.Errorf("evaluating data size at program counter: %w", err)
		}
	}
	return nil
}

func assignInstructionAddress(currentScope *scope.Scope, asm *Assembler, n *instruction, programCounter uint64) (uint64, error) {
	n.address = programCounter

	name := n.name
	ins, ok := asm.cfg.Arch.Instructions[name]
	if !ok {
		return 0, fmt.Errorf("unsupported instruction '%s'", name)
	}

	// handle disambiguous addressing mode to reduce absolute addressings to
	// zeropage ones if the used address value fits into byte
	switch n.addressing {
	case ast.XAddressing:
		value, err := getArgumentValue(n.argument, currentScope)
		if err != nil {
			return 0, fmt.Errorf("getting instruction argument: %w", err)
		}
		if value > math.MaxUint8 {
			n.addressing = AbsoluteXAddressing
		} else {
			n.addressing = ZeroPageXAddressing
		}

	case ast.YAddressing:
		value, err := getArgumentValue(n.argument, currentScope)
		if err != nil {
			return 0, fmt.Errorf("getting instruction argument: %w", err)
		}
		if value > math.MaxUint8 {
			n.addressing = AbsoluteYAddressing
		} else {
			n.addressing = ZeroPageYAddressing
		}
	}

	addressingInfo, ok := ins.Addressing[n.addressing]
	if !ok {
		return 0, fmt.Errorf("unsupported instruction '%s' addressing %d", name, n.addressing)
	}

	programCounter += uint64(addressingInfo.Size)
	return programCounter, nil
}
