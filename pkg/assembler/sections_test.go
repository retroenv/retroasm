package assembler

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/assert"
)

func TestSectionsIncludeReservationsAndGapsButExcludeEnums(t *testing.T) {
	configuration := &config.Segment{
		Memory: config.Memory{
			Name: "RAM",
		},
		SegmentName:  "DATA",
		SegmentStart: 0x200,
	}
	asm := &Assembler[any]{
		cfg: &config.Config[any]{
			SegmentsOrdered: []*config.Segment{configuration},
		},
		segments: map[string]*segment{
			"DATA": {
				config: configuration,
				nodes: []ast.Node{
					&variable{
						address: 0x200,
						v:       ast.NewVariable("scratch", 4),
					},
					&instruction{
						address: 0x210,
						opcodes: []byte{0xa9, 42},
					},
					ast.Enum{},
					&variable{
						address: 0x700,
						v:       ast.NewVariable("constant", 40),
					},
					ast.EnumEnd{},
				},
			},
		},
	}
	assert.Equal(t, []Section{{
		Name:   "DATA",
		Memory: "RAM",
		Start:  0x200,
		End:    0x212,
		Bytes:  2,
	}}, asm.Sections())
}
