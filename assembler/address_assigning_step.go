package assembler

import (
	"errors"
	"fmt"
	"math"

	"github.com/retroenv/retroasm/arch"
	"github.com/retroenv/retroasm/parser/ast"
	"github.com/retroenv/retroasm/scope"
	. "github.com/retroenv/retrogolib/addressing"
)

type addressAssign struct {
	arch arch.Architecture

	currentScope   *scope.Scope // current scope, can be a function scope with file scope as parent
	programCounter uint64

	enumActive               bool
	enumBackupProgramCounter uint64
}

// assignAddressesStep assigns an address for every node in each scope.
func assignAddressesStep(asm *Assembler) error {
	var err error
	aa := addressAssign{
		arch:         asm.cfg.Arch,
		currentScope: asm.fileScope,
	}

	for _, seg := range asm.segmentsOrder {
		aa.programCounter = seg.config.Start

		for _, node := range seg.nodes {
			switch n := node.(type) {
			case ast.Base:
				aa.programCounter, err = assignBaseAddress(n)

			case ast.Configuration:

			case ast.Enum:
				aa.programCounter, err = assignEnumAddress(&aa, n)

			case ast.EnumEnd:
				aa.programCounter, err = assignEnumEndAddress(&aa)

			case *data:
				aa.programCounter, err = assignDataAddress(aa, n)

			case *instruction:
				aa.programCounter, err = assignInstructionAddress(aa, n)

			case scopeChange:
				aa.currentScope = n.scope

			case *scope.Symbol:
				err = assignSymbolAddress(aa, n)

			case *variable:
				assignVariableAddress(aa, n)

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

func assignDataAddress(aa addressAssign, d *data) (uint64, error) {
	if d.size.IsEvaluatedAtAddressAssign() {
		_, err := d.size.EvaluateAtProgramCounter(aa.currentScope, d.width, aa.programCounter)
		if err != nil {
			return 0, fmt.Errorf("evaluating data size at program counter: %w", err)
		}
	}

	d.address = aa.programCounter
	size, err := d.size.IntValue()
	if err != nil {
		return 0, fmt.Errorf("getting data node size: %w", err)
	}
	aa.programCounter += uint64(size)
	return aa.programCounter, nil
}

func assignVariableAddress(aa addressAssign, v *variable) uint64 {
	v.address = aa.programCounter
	aa.programCounter += uint64(v.v.Size)
	return aa.programCounter
}

func assignBaseAddress(b ast.Base) (uint64, error) {
	i, err := b.Address.IntValue()
	if err != nil {
		return 0, fmt.Errorf("getting base node address: %w", err)
	}
	return uint64(i), nil
}

func assignSymbolAddress(aa addressAssign, sym *scope.Symbol) error {
	sym.SetAddress(aa.programCounter)
	exp := sym.Expression()
	if exp != nil && exp.IsEvaluatedAtAddressAssign() {
		_, err := exp.EvaluateAtProgramCounter(aa.currentScope, aa.arch.AddressWidth, aa.programCounter)
		if err != nil {
			return fmt.Errorf("evaluating data size at program counter: %w", err)
		}
	}
	return nil
}

func assignInstructionAddress(aa addressAssign, n *instruction) (uint64, error) {
	n.address = aa.programCounter

	name := n.name
	ins, ok := aa.arch.Instructions[name]
	if !ok {
		return 0, fmt.Errorf("unsupported instruction '%s'", name)
	}

	// handle disambiguous addressing mode to reduce absolute addressings to
	// zeropage ones if the used address value fits into byte
	switch n.addressing {
	case ast.XAddressing:
		value, err := getArgumentValue(n.argument, aa.currentScope)
		if err != nil {
			return 0, fmt.Errorf("getting instruction argument: %w", err)
		}
		if value > math.MaxUint8 {
			n.addressing = AbsoluteXAddressing
		} else {
			n.addressing = ZeroPageXAddressing
		}

	case ast.YAddressing:
		value, err := getArgumentValue(n.argument, aa.currentScope)
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

	programCounter := aa.programCounter + uint64(addressingInfo.Size)
	return programCounter, nil
}

func assignEnumAddress(aa *addressAssign, e ast.Enum) (uint64, error) {
	if aa.enumActive {
		return 0, errors.New("invalid enum inside enum context")
	}

	aa.enumBackupProgramCounter = aa.programCounter
	aa.enumActive = true

	pc, err := e.Address.IntValue()
	if err != nil {
		return 0, fmt.Errorf("getting enum address: %w", err)
	}
	return uint64(pc), nil
}

func assignEnumEndAddress(aa *addressAssign) (uint64, error) {
	if !aa.enumActive {
		return 0, errors.New("enum outside of enum context")
	}

	aa.enumActive = false

	return aa.enumBackupProgramCounter, nil
}
