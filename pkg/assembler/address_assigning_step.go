package assembler

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retroasm/pkg/scope"
)

type addressAssign[T any] struct {
	arch arch.Architecture[T]

	currentScope   *scope.Scope // current scope, can be a function scope with file scope as parent
	programCounter uint64

	enumActive               bool
	enumBackupProgramCounter uint64
}

// ArgumentValue returns the value of an instruction argument, either a number or a symbol value.
func (aa *addressAssign[T]) ArgumentValue(argument any) (uint64, error) {
	switch arg := argument.(type) {
	case uint64:
		return arg, nil

	case ast.Number:
		return arg.Value, nil

	case reference:
		name, offset := parseReferenceOffset(arg.name)

		sym, err := aa.currentScope.GetSymbol(name)
		if err != nil {
			return 0, fmt.Errorf("getting instruction argument: %w", err)
		}

		value, err := sym.Value(aa.currentScope)
		if err != nil {
			return 0, fmt.Errorf("getting symbol '%s' value: %w", name, err)
		}

		switch v := value.(type) {
		case int64:
			adjusted, err := applyInt64Offset(v, offset)
			if err != nil {
				return 0, err
			}
			if adjusted < 0 {
				return 0, fmt.Errorf("reference '%s' resolved to negative value %d", arg.name, adjusted)
			}
			return uint64(adjusted), nil
		case uint64:
			return applyUint64Offset(v, offset)
		default:
			return 0, fmt.Errorf("unexpected argument value type %T", value)
		}

	case ast.Label:
		return aa.ArgumentValue(reference{name: arg.Name})

	case ast.Identifier:
		return aa.ArgumentValue(reference{name: arg.Name})

	case ast.Expression:
		return aa.argumentExpressionValue(arg)

	default:
		return 0, fmt.Errorf("unexpected argument type %T", arg)
	}
}

// RelativeOffset returns the relative offset between two addresses.
func (aa *addressAssign[T]) RelativeOffset(destination, addressAfterInstruction uint64) (byte, error) {
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

// ProgramCounter returns the current program counter.
func (aa *addressAssign[T]) ProgramCounter() uint64 {
	return aa.programCounter
}

func (aa *addressAssign[T]) argumentExpressionValue(exprNode ast.Expression) (uint64, error) {
	if exprNode.Value == nil {
		return 0, errors.New("expression argument value is nil")
	}

	width := aa.addressWidth()

	var (
		value any
		err   error
	)

	if exprNode.Value.IsEvaluatedAtAddressAssign() {
		value, err = exprNode.Value.EvaluateAtProgramCounter(aa.currentScope, width, aa.programCounter)
	} else {
		value, err = exprNode.Value.Evaluate(aa.currentScope, width)
	}
	if err != nil {
		return 0, fmt.Errorf("evaluating expression argument: %w", err)
	}

	switch v := value.(type) {
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("expression result %d is negative", v)
		}
		return uint64(v), nil
	case uint64:
		return v, nil
	default:
		return 0, fmt.Errorf("unexpected expression result type %T", value)
	}
}

func (aa *addressAssign[T]) addressWidth() int {
	if aa.arch == nil {
		return 16
	}
	return aa.arch.AddressWidth()
}

// assignAddressesStep assigns an address for every node in each scope.
func assignAddressesStep[T any](asm *Assembler[T]) error {
	var err error
	aa := addressAssign[T]{
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
				aa.programCounter, err = aa.arch.AssignInstructionAddress(&aa, n)

			case scopeChange:
				aa.currentScope = n.scope

			case *symbol:
				err = assignSymbolAddress(aa, n)

			case *variable:
				aa.programCounter = assignVariableAddress(aa, n)

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

// parseReferenceOffset splits a reference name into a base symbol name and
// an integer offset. It handles names like "symbol+8" or "symbol-3".
// If no offset is present, offset is 0.
func parseReferenceOffset(name string) (string, int64) {
	if idx := strings.LastIndexAny(name, "+-"); idx > 0 {
		offsetStr := name[idx:]
		if n, err := strconv.ParseInt(offsetStr, 10, 64); err == nil {
			return name[:idx], n
		}
	}
	return name, 0
}

func applyInt64Offset(base, offset int64) (int64, error) {
	if (offset > 0 && base > math.MaxInt64-offset) ||
		(offset < 0 && base < math.MinInt64-offset) {

		return 0, fmt.Errorf("offset overflow for base %d and offset %d", base, offset)
	}

	return base + offset, nil
}

func applyUint64Offset(base uint64, offset int64) (uint64, error) {
	if offset >= 0 {
		delta := uint64(offset)
		if base > math.MaxUint64-delta {
			return 0, fmt.Errorf("offset overflow for base %d and offset %d", base, offset)
		}
		return base + delta, nil
	}

	delta := uint64(-offset)
	if base < delta {
		return 0, fmt.Errorf("offset underflow for base %d and offset %d", base, offset)
	}

	return base - delta, nil
}

func assignDataAddress[T any](aa addressAssign[T], d *data) (uint64, error) {
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

func assignVariableAddress[T any](aa addressAssign[T], v *variable) uint64 {
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

func assignSymbolAddress[T any](aa addressAssign[T], sym *symbol) error {
	sym.SetAddress(aa.programCounter)
	exp := sym.Expression()
	if exp != nil && exp.IsEvaluatedAtAddressAssign() {
		_, err := exp.EvaluateAtProgramCounter(aa.currentScope, aa.arch.AddressWidth(), aa.programCounter)
		if err != nil {
			return fmt.Errorf("evaluating data size at program counter: %w", err)
		}
	}
	return nil
}

func assignEnumAddress[T any](aa *addressAssign[T], e ast.Enum) (uint64, error) {
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

func assignEnumEndAddress[T any](aa *addressAssign[T]) (uint64, error) {
	if !aa.enumActive {
		return 0, errors.New("enum end outside of enum context")
	}

	aa.enumActive = false

	return aa.enumBackupProgramCounter, nil
}
