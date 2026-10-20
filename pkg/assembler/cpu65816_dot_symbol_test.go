package assembler

import (
	"bytes"
	"strings"
	"testing"

	asmcpu65816 "github.com/retroenv/retroasm/pkg/arch/cpu65816"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retrogolib/assert"
)

func TestAssemblerCPU65816LiteralDotSymbolDefinitionAndReference(t *testing.T) {
	const source = `.segment "CODE"
main:
lda f:.Lretrogo_rodata_table,x
jmp .Ldone
.Lretrogo_rodata_table:
.db 3,9
.Ldone:
rts
`
	for _, compat := range []config.CompatibilityMode{config.CompatDefault, config.CompatNesasm} {
		cfg := asmcpu65816.New()
		cfg.CompatibilityMode = compat
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
		assert.Equal(t, []byte{0xbf, 0x07, 0x02, 0x00, 0x4c, 0x09, 0x02, 3, 9, 0x60}, binary.Bytes())
		prefix := ""
		if compat == config.CompatNesasm {
			prefix = "main"
		}
		assert.Equal(t, uint64(0x207), asm.Symbols()[prefix+".Lretrogo_rodata_table"])
		assert.Equal(t, uint64(0x209), asm.Symbols()[prefix+".Ldone"])
	}
}
