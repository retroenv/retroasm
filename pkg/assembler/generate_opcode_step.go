package assembler

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/retroenv/retroasm/pkg/number"
	"github.com/retroenv/retroasm/pkg/scope"
)

// generateOpcodesStep generates the opcodes for instructions and data nodes and resolves any
// references to their value or assigned addresses.
func generateOpcodesStep[T any](_ context.Context, asm *Assembler[T]) error {
	asm.instructionRelocations = nil

	currentScope := asm.fileScope
	arch := asm.cfg.Arch

	for _, seg := range asm.segmentsOrder {
		for _, node := range seg.nodes {
			switch n := node.(type) {
			case *data:
				if err := generateDeferredDataBytes(currentScope, n, asm.byteOrder); err != nil {
					return fmt.Errorf("generating deferred data at $%x: %w", n.address, err)
				}
				if err := generateReferenceDataBytes(currentScope, n, asm.byteOrder); err != nil {
					return fmt.Errorf("generating data node opcode: %w", err)
				}
				if n.fill {
					if err := generateDataFillBytes(n); err != nil {
						return fmt.Errorf("generating data node opcode: %w", err)
					}
				}

			case *instruction:
				assigner := &addressAssign[T]{
					arch:                   arch,
					currentScope:           currentScope,
					programCounter:         n.Address(),
					instructionRelocations: &asm.instructionRelocations,
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

func generateDeferredDataBytes(currentScope *scope.Scope, dat *data, order binary.ByteOrder) error {
	if !dat.deferred {
		return nil
	}

	// Address assignment has completed, so forward symbols now have values.
	dat.values = nil
	for index, item := range dat.expressions {
		value, err := item.Evaluate(currentScope, dat.width)
		if err != nil {
			return fmt.Errorf("evaluating deferred data item %d: %w", index, err)
		}
		if err := appendDataExpressionValue(dat, value, order); err != nil {
			return fmt.Errorf("appending deferred data item %d: %w", index, err)
		}
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
func generateReferenceDataBytes(currentScope *scope.Scope, d *data, order binary.ByteOrder) error {
	for i, item := range d.values {
		ref, ok := item.(reference)
		if !ok {
			continue
		}

		address, err := resolveReferenceAddress(currentScope, ref)
		if err != nil {
			return err
		}

		var b []byte

		switch ref.typ {
		case fullAddress:
			b, err = number.WriteToBytesWithOrder(address, d.width, order)
			if err != nil {
				return fmt.Errorf("writing full address as bytes: %w", err)
			}
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

func resolveReferenceAddress(currentScope *scope.Scope, ref reference) (uint64, error) {
	sym, err := currentScope.GetSymbol(ref.name)
	if err != nil {
		return 0, fmt.Errorf("getting data reference: %w", err)
	}

	value, err := sym.Value(currentScope)
	if err != nil {
		return 0, fmt.Errorf("getting symbol '%s' value: %w", ref.name, err)
	}

	switch v := value.(type) {
	case int64:
		adjusted, offsetErr := applyInt64Offset(v, ref.offset)
		if offsetErr != nil {
			return 0, fmt.Errorf("applying reference offset: %w", offsetErr)
		}
		if adjusted < 0 {
			return 0, fmt.Errorf("reference '%s' resolved to negative value %d", ref.name, adjusted)
		}
		return uint64(adjusted), nil
	case uint64:
		adjusted, offsetErr := applyUint64Offset(v, ref.offset)
		if offsetErr != nil {
			return 0, fmt.Errorf("applying reference offset: %w", offsetErr)
		}
		return adjusted, nil
	default:
		return 0, fmt.Errorf("unexpected reference value type %T", value)
	}
}
