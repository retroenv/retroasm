package assembler

import (
	"bytes"
	"strings"
	"testing"

	"github.com/retroenv/retroasm/pkg/arch/cpu6502"
	"github.com/retroenv/retrogolib/assert"
)

func TestBankedOutputPadsIndependentMemoryAreas(t *testing.T) {
	cfg := cpu6502.New()
	assert.NoError(t, cfg.ReadCa65Config(strings.NewReader(`
MEMORY {
    FIXED: start = $8000, size = $10, fill = yes, fillval = $ff;
    BANK0: start = $c000, size = $10, fill = yes, fillval = $a5;
    BANK1: start = $c000, size = $10, fill = yes, fillval = $5a;
}

SEGMENTS {
    CODE: load = FIXED;
    COLD0: load = BANK0;
    COLD1: load = BANK1;
}`)))
	var buf bytes.Buffer
	asm := New(cfg, &buf)
	assert.NoError(t, asm.Process(t.Context(), strings.NewReader(`
.segment "CODE"
entry: jsr first
       jsr second
       rts
.segment "COLD0"
first: lda #1
       rts
.segment "COLD1"
second: lda #2
        rts
`)))
	want := append(bytes.Repeat([]byte{0xff}, 16), bytes.Repeat([]byte{0xa5}, 16)...)
	want = append(want, bytes.Repeat([]byte{0x5a}, 16)...)
	copy(want, []byte{0x20, 0, 0xc0, 0x20, 0, 0xc0, 0x60})
	copy(want[16:], []byte{0xa9, 1, 0x60})
	copy(want[32:], []byte{0xa9, 2, 0x60})
	assert.Equal(t, want, buf.Bytes())
	assert.Equal(t, uint64(0xc000), asm.Symbols()["first"])
	assert.Equal(t, uint64(0xc000), asm.Symbols()["second"])
}

func TestSegmentStartSetsLabelAndOutputAddress(t *testing.T) {
	cfg := cpu6502.New()
	assert.NoError(t, cfg.ReadCa65Config(strings.NewReader(`
MEMORY { PRG: start = $8000, size = $20, fill = yes, fillval = $ff; }
SEGMENTS {
    CODE: load = PRG;
    OTHER: load = PRG, start = $8010;
}`)))
	var buf bytes.Buffer
	asm := New(cfg, &buf)
	assert.NoError(t, asm.Process(t.Context(), strings.NewReader(`
.segment "CODE"
    jsr target
.segment "OTHER"
target: rts
`)))
	want := bytes.Repeat([]byte{0xff}, 32)
	copy(want, []byte{0x20, 0x10, 0x80})
	want[16] = 0x60
	assert.Equal(t, want, buf.Bytes())
	assert.Equal(t, uint64(0x8010), asm.Symbols()["target"])
}
