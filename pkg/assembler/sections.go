package assembler

import "github.com/retroenv/retroasm/pkg/parser/ast"

// Section reports the assigned extent and encoded bytes of one assembled segment.
// End is exclusive. Bytes excludes address gaps, reservations, and memory-area fill.
type Section struct {
	Name   string
	Memory string
	Start  uint64
	End    uint64
	Bytes  uint64
}

// Sections returns independent layout records in configured output order.
// Call it after successful assembly, when addresses and encodings are final.
func (asm *Assembler[T]) Sections() []Section {
	sections := make([]Section, 0, len(asm.segments))
	for _, configured := range asm.cfg.SegmentsOrdered {
		segment, found := asm.segments[configured.SegmentName]
		if !found {
			continue
		}
		section := segment.layout()
		sections = append(sections, section)
	}

	return sections
}

func (seg *segment) layout() Section {
	section := Section{
		Name:   seg.config.SegmentName,
		Memory: seg.config.Memory.Name,
		Start:  seg.config.SegmentStart,
		End:    seg.config.SegmentStart,
	}
	hasExtent := false
	enumDepth := 0
	for _, node := range seg.nodes {
		switch node.(type) {
		case ast.Enum:
			enumDepth++
			continue

		case ast.EnumEnd:
			enumDepth--
			continue

		case *variable:
			if enumDepth > 0 {
				continue
			}
		}

		start, extent, encoded := assembledNodeExtent(node)
		section.Bytes += encoded
		if extent == 0 {
			continue
		}
		if !hasExtent {
			section.Start = start
			section.End = start
			hasExtent = true
		}

		section.Start = min(section.Start, start)
		section.End = max(section.End, start+extent)
	}

	return section
}

func assembledNodeExtent(node ast.Node) (uint64, uint64, uint64) {
	switch value := node.(type) {
	case *instruction:
		size := uint64(len(value.opcodes))
		return value.address, size, size

	case *data:
		var size uint64
		for _, entry := range value.values {
			if encoded, ok := entry.([]byte); ok {
				size += uint64(len(encoded))
			}
		}

		return value.address, size, size

	case *variable:
		return value.address, uint64(value.v.Size), 0
	default:
		return 0, 0, 0
	}
}
