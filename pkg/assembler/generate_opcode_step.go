package assembler

import (
	"context"
	"fmt"

	"github.com/retroenv/retroasm/pkg/scope"
)

// generateOpcodesStep generates the opcodes for instructions and data nodes and resolves any
// references to their value or assigned addresses.
func generateOpcodesStep[T any](_ context.Context, asm *Assembler[T]) error {
	currentScope := asm.fileScope
	arch := asm.cfg.Arch

	for _, seg := range asm.segmentsOrder {
		for _, node := range seg.nodes {
			switch n := node.(type) {
			case *data:
				if err := generateDeferredDataBytes(currentScope, n); err != nil {
					return fmt.Errorf("generating deferred data at $%x: %w", n.address, err)
				}
				if err := generateReferenceDataBytes(currentScope, n); err != nil {
					return fmt.Errorf("generating data node opcode: %w", err)
				}
				if n.fill {
					if err := generateDataFillBytes(n); err != nil {
						return fmt.Errorf("generating data node opcode: %w", err)
					}
				}

			case *instruction:
				assigner := &addressAssign[T]{
					arch:           arch,
					currentScope:   currentScope,
					programCounter: n.Address(),
				}
				if err := arch.GenerateInstructionOpcode(assigner, n); err != nil {
					return fmt.Errorf("generating instruction '%s' at $%x opcode: %w", n.Name(), n.Address(), err)
				}

			case scopeChange:
				currentScope = n.scope
			}
		}
	}
	return nil
}

func generateDeferredDataBytes(currentScope *scope.Scope, dat *data) error {
	if !dat.deferred {
		return nil
	}

	// Address assignment has completed, so forward symbols now have values.
	value, err := dat.expression.Evaluate(currentScope, dat.width)
	if err != nil {
		return fmt.Errorf("evaluating deferred data expression: %w", err)
	}
	dat.values = nil
	if err := appendDataExpressionValue(dat, value); err != nil {
		return err
	}

	actualSize := 0
	for _, value := range dat.values {
		bytes, ok := value.([]byte)
		if !ok {
			return fmt.Errorf("unsupported deferred data value type %T", value)
		}
		actualSize += len(bytes)
	}
	// A changed byte count would invalidate every address assigned after this node.
	if actualSize != dat.deferredSize {
		return fmt.Errorf("deferred data size changed from %d to %d bytes", dat.deferredSize, actualSize)
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
		case bankAddressByte:
			b = []byte{byte(address >> 16)}
		default:
			return fmt.Errorf("unsupported reference type %d", ref.typ)
		}

		d.values[i] = b
	}
	return nil
}
