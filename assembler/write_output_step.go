package assembler

import (
	"fmt"

	"github.com/retroenv/retroasm/assembler/config"
)

// writeOutputStep writes the filled memory segments to the output stream.
func writeOutputStep(asm *Assembler) error {
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

		if uint64(len(mem.data))-mem.start > mem.size {
			return fmt.Errorf("memory '%s' exceeds size limit %d, %d bytes written",
				memName, mem.size, len(mem.data))
		}

		buf := mem.data[mem.start:]
		_, err = asm.writer.Write(buf)
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
					mem.write(b, offset, seg.config.SegmentStart)
					offset += uint64(len(b))
				}

			case *instruction:
				mem.write(n.opcodes, n.address, seg.config.SegmentStart)
			}
		}
	}

	return memories, nil
}
