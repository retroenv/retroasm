package assembler

import (
	"bytes"
	"strings"
	"testing"

	asmcpu68000 "github.com/retroenv/retroasm/pkg/arch/cpu68000"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retrogolib/assert"
)

func TestAssemblerCPU68000LiteralDotSymbolLookup(t *testing.T) {
	const source = `.segment "CODE"
main:
lea .Lretrogo_rodata_table.l,a0
move.b 0(a0,d0.w),d0
jmp .Ldone.l
.Lretrogo_rodata_table:
.db 3,9
.Ldone:
rts
`
	binary, symbols := assembleCPU68000DotSource(t, source, 0x120000)
	assert.Equal(t, []byte{0x41, 0xf9, 0, 0x12, 0, 0x10, 0x10, 0x30, 0, 0, 0x4e, 0xf9, 0, 0x12, 0, 0x12, 3, 9, 0x4e, 0x75}, binary)
	assert.Equal(t, uint64(0x120010), symbols[".Lretrogo_rodata_table"])
	assert.Equal(t, uint64(0x120012), symbols[".Ldone"])
}

func TestAssemblerCPU68000MotorolaDataAndAlignment(t *testing.T) {
	const source = `ORG $0
DC.L $FFFE00,main
ORG $200
main:
lea .Lretrogo_rodata_table.l,a0
rts
DC.B $A5
DCB.B (8-(*%8))%8,0
.Lretrogo_rodata_table:
DC.B 3,9
DC.W $1234
DC.L .Lretrogo_rodata_table
`
	binary, symbols := assembleCPU68000DotSource(t, source, 0)
	assert.Equal(t, []byte{0, 0xff, 0xfe, 0, 0, 0, 2, 0}, binary[:8])
	assert.Equal(t, uint64(0x200), symbols["main"])
	assert.Equal(t, uint64(0x210), symbols[".Lretrogo_rodata_table"])
	assert.Equal(t, []byte{0xa5, 0, 0, 0, 0, 0, 0, 0, 3, 9, 0x12, 0x34, 0, 0, 2, 0x10}, binary[0x208:])
}

func TestAssemblerCPU68000MotorolaBlockWidths(t *testing.T) {
	const source = `DCB.B 2*3,$7F
DCB.W 2,$1234
DCB.L 2,$12345678
DCB.B 0,99
DCB.B 2
DCB 1,2
`
	binary, _ := assembleCPU68000DotSource(t, source, 0)
	assert.Equal(t, []byte{
		0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f,
		0x12, 0x34, 0x12, 0x34,
		0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78,
		0, 0, 1, 2,
	}, binary)
}

func TestAssemblerCPU68000InvalidDataAndDotOperands(t *testing.T) {
	for _, source := range []string{
		"lea .1.l,a0", "move.l #.1,d0", "lea .Lmissing.l,a0",
		"DC.Q 1", "DCB.Q 1", "DCB.B 1,2,3", "DC.B", "DCB.B",
	} {
		t.Run(source, func(t *testing.T) {
			var binary bytes.Buffer
			asm := New(asmcpu68000.New(), &binary)
			assert.Error(t, asm.Process(t.Context(), strings.NewReader(source)))
		})
	}
}

func assembleCPU68000DotSource(t *testing.T, source string, start uint64) ([]byte, map[string]uint64) {
	t.Helper()
	cfg := asmcpu68000.New()
	segment := &config.Segment{
		Memory: config.Memory{
			Start: start,
			Size:  0x1000,
		},
		SegmentName:  "CODE",
		SegmentStart: start,
	}
	cfg.Segments = map[string]*config.Segment{"CODE": segment}
	cfg.SegmentsOrdered = []*config.Segment{segment}
	var binary bytes.Buffer
	asm := New(cfg, &binary)
	assert.NoError(t, asm.Process(t.Context(), strings.NewReader(source)))
	return binary.Bytes(), asm.Symbols()
}
