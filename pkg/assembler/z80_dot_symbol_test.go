package assembler

import (
	"bytes"
	"strings"
	"testing"

	asmz80 "github.com/retroenv/retroasm/pkg/arch/z80"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retrogolib/assert"
)

func TestAssemblerZ80LiteralDotSymbolDefinitionAndReference(t *testing.T) {
	const source = `.segment "CODE"
main:
ld hl,.Ltable
ld a,(.Ltable+1)
jp .Ldone
.Ltable:
.db 3,9
.Ldone:
ret
`
	cfg := asmz80.New()
	segment := &config.Segment{
		Memory: config.Memory{
			Start: 0x200,
			Size:  0x100,
		},
		SegmentName:  "CODE",
		SegmentStart: 0x200,
	}
	cfg.Segments = map[string]*config.Segment{"CODE": segment}
	cfg.SegmentsOrdered = []*config.Segment{segment}
	var binary bytes.Buffer
	asm := New(cfg, &binary)
	assert.NoError(t, asm.Process(t.Context(), strings.NewReader(source)))
	assert.Equal(t, []byte{0x21, 0x09, 0x02, 0x3a, 0x0a, 0x02, 0xc3, 0x0b, 0x02, 3, 9, 0xc9}, binary.Bytes())
	assert.Equal(t, uint64(0x209), asm.Symbols()[".Ltable"])
	assert.Equal(t, uint64(0x20b), asm.Symbols()[".Ldone"])
}
