package assembler

import (
	"fmt"

	"github.com/retroenv/assembler/parser/ast"
	"github.com/retroenv/assembler/scope"
)

// writeOutputStep writes the filled memory segments to the output stream.
func writeOutputStep(asm *Assembler) error {
	memories, err := writeSegmentsToMemory(asm, asm.segments)
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

		if len(mem.data) > int(mem.size) {
			return fmt.Errorf("memory '%s' exceeds size limit %d, %d bytes written",
				memName, mem.size, len(mem.data))
		}

		_, err = asm.writer.Write(mem.data)
		if err != nil {
			return fmt.Errorf("writing fill data to output: %w", err)
		}

		delete(memories, memName)
	}

	return nil
}

func writeSegmentsToMemory(asm *Assembler, segments map[string]*segment) (map[string]*memory, error) {
	memories := map[string]*memory{}

	for _, segOrdered := range asm.cfg.SegmentsOrdered {
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

		offset := seg.config.SegmentStart

		for _, node := range seg.nodes {
			switch n := node.(type) {
			case *data:
				for _, val := range n.values {
					b, ok := val.([]byte)
					if !ok {
						return nil, fmt.Errorf("unsupported node value type %T", val)
					}
					mem.write(b, offset)
					offset += uint64(len(b))
				}

			case *scope.Symbol,
				*base,
				*ast.Configuration,
				scopeChange:

			case *instruction:
				mem.write(n.opcodes, offset)
				offset += uint64(len(n.opcodes))

			case *variable:
				offset += uint64(n.v.Size)

			default:
				return nil, fmt.Errorf("unsupported node type %T", n)
			}
		}
	}

	return memories, nil
}
