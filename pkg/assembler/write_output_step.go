package assembler

import (
	"context"
	"fmt"

	"github.com/retroenv/retroasm/pkg/assembler/config"
)

// writeOutputStep writes the filled memory segments to the output stream.
func writeOutputStep[T any](_ context.Context, asm *Assembler[T]) error {
	memories, err := writeSegmentsToMemory(asm.cfg.SegmentsOrdered, asm.segments)
	if err != nil {
		return fmt.Errorf("writing segments to memory: %w", err)
	}

	for _, segOrdered := range asm.cfg.SegmentsOrdered {
		seg, ok := asm.segments[segOrdered.SegmentName]
		if !ok {
			continue
		}

		memName := seg.config.Memory.Name
		mem, ok := memories[memName]
		if !ok {
			// has already been processed due to reference from another segment
			continue
		}

		_, err = asm.writer.Write(mem.data)
		if err != nil {
			return fmt.Errorf("writing fill data to output: %w", err)
		}

		delete(memories, memName)
	}

	return nil
}

func writeSegmentsToMemory(configSegmentsOrdered []*config.Segment,
	segments map[string]*segment) (map[string]*memory, error) {

	memories := map[string]*memory{}

	for _, segOrdered := range configSegmentsOrdered {
		seg, ok := segments[segOrdered.SegmentName]
		if !ok {
			continue
		}

		memName := seg.config.Memory.Name
		mem, ok := memories[memName]
		if !ok {
			mem = newMemory(seg.config.Memory)
			memories[memName] = mem
		}

		for _, node := range seg.nodes {
			switch n := node.(type) {
			case *data:
				offset := n.address
				for _, val := range n.values {
					b, ok := val.([]byte)
					if !ok {
						return nil, fmt.Errorf("unsupported node value type %T", val)
					}
					if err := mem.write(b, offset); err != nil {
						return nil, fmt.Errorf("memory %q: %w", memName, err)
					}
					offset += uint64(len(b))
				}

			case *instruction:
				if err := mem.write(n.opcodes, n.address); err != nil {
					return nil, fmt.Errorf("memory %q: %w", memName, err)
				}
			}
		}
	}

	return memories, nil
}
