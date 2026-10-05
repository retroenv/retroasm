package cpu65816

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/retroenv/retroasm/pkg/arch/cpu65816/parser"
	"github.com/retroenv/retroasm/pkg/assembler"
	"github.com/retroenv/retroasm/pkg/codec"
	"github.com/retroenv/retrogolib/assert"
)

func TestForwardRAMReferencesRetainAbsoluteFormsWithoutLoadBytes(t *testing.T) {
	const source = `.segment "CODE"
lda home
sta home+1
ldx home
ldy home
lda home,x
lda home,y
.segment "RAM"
home: .res 8
`
	for _, address := range []uint64{0x20, 0x100} {
		for _, typed := range []bool{false, true} {
			t.Run(fmt.Sprintf("address=%x/typed=%t", address, typed), func(t *testing.T) {
				cfg := New()
				layout := fmt.Sprintf(`MEMORY {
ROM: start = $8000, size = $8000;
WRAM: start = $%x, size = $100;
}
SEGMENTS {
CODE: load = ROM, type = ro;
RAM: load = WRAM, type = bss;
}`, address)
				assert.NoError(t, cfg.ReadCa65Config(strings.NewReader(layout)))
				var output bytes.Buffer
				asm := assembler.New(cfg, &output)
				if typed {
					parserCodec, err := codec.New(cfg)
					assert.NoError(t, err)
					stream, err := codec.ParseStreamWithState(t.Context(), parserCodec, "forward.asm", strings.NewReader(source), parser.DefaultState())
					assert.NoError(t, err)
					result, err := parserCodec.AssembleStream(t.Context(), stream)
					assert.NoError(t, err)
					_, err = output.Write(result.Binary)
					assert.NoError(t, err)
				} else {
					assert.NoError(t, asm.Process(t.Context(), strings.NewReader(source)))
				}
				low, high := byte(address), byte(address>>8)
				assert.Equal(t, []byte{
					0xad, low, high, 0x8d, low + 1, high, 0xae, low, high,
					0xac, low, high, 0xbd, low, high, 0xb9, low, high,
				}, output.Bytes())
			})
		}
	}
}
