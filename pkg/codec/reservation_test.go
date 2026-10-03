package codec_test

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	asmchip8 "github.com/retroenv/retroasm/pkg/arch/chip8"
	asmcpu6502 "github.com/retroenv/retroasm/pkg/arch/cpu6502"
	asmcpu65816 "github.com/retroenv/retroasm/pkg/arch/cpu65816"
	asmcpu68000 "github.com/retroenv/retroasm/pkg/arch/cpu68000"
	asmsm83 "github.com/retroenv/retroasm/pkg/arch/sm83"
	asmz80 "github.com/retroenv/retroasm/pkg/arch/z80"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/codec"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/cpu6502"
	"github.com/retroenv/retrogolib/assert"
)

func TestCodec_ReservationAssignsRAMWithoutLoadBytes(t *testing.T) {
	t.Parallel()

	// The assembler discarded reservations and assigned each RAM object the same address.
	c := newReservationCodec(t, 0x200, 8)
	stream, err := c.ParseStream(t.Context(), "storage.asm", strings.NewReader(`
.segment "RAM"
first:
.res 3
second:
.res 5
end:
.segment "CODE"
.addr first, second, end
`))
	assert.NoError(t, err)

	for _, route := range []string{"typed", "text"} {
		t.Run(route, func(t *testing.T) {
			t.Parallel()

			input := stream
			if route == "text" {
				text, formatErr := c.FormatStream(stream)
				assert.NoError(t, formatErr)
				parsed, parseErr := c.ParseStream(t.Context(), "roundtrip.asm", strings.NewReader(text))
				assert.NoError(t, parseErr)
				input = parsed
			}

			assembly, assembleErr := c.AssembleStream(t.Context(), input)
			assert.NoError(t, assembleErr)
			assert.Equal(t, []byte{0, 2, 3, 2, 8, 2}, assembly.Binary)
			assert.Equal(t, uint64(0x200), assembly.Symbols["first"])
			assert.Equal(t, uint64(0x203), assembly.Symbols["second"])
			assert.Equal(t, uint64(0x208), assembly.Symbols["end"])
			assert.NoError(t, assembly.Stream.Validate())
		})
	}
}

func TestCodec_ReservationRejectsInvalidExtent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		start uint64
		size  uint64
		first int
		last  int
	}{
		{
			name:  "one byte above RAM capacity",
			start: 0x200,
			size:  8,
			first: 3,
			last:  6,
		},
		{
			name:  "negative typed reservation",
			start: 0x200,
			size:  8,
			first: -1,
			last:  1,
		},
		{
			name:  "address width overflow",
			start: 0xffff,
			size:  2,
			first: 2,
		},
		{
			name:  "unsigned address overflow",
			start: math.MaxUint64,
			size:  math.MaxUint64,
			first: 2,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			c := newReservationCodec(t, test.start, test.size)
			assembly, err := c.Assemble(t.Context(), []ast.Node{
				ast.NewSegment("RAM"),
				ast.NewVariable("", test.first),
				ast.NewVariable("", test.last),
			})
			assert.Error(t, err)
			assert.Nil(t, assembly)
		})
	}
}

func TestCodec_ReservationExactAddressEnd(t *testing.T) {
	t.Parallel()

	c := newReservationCodec(t, 0xffff, 1)
	assembly, err := c.Assemble(t.Context(), []ast.Node{
		ast.NewSegment("RAM"),
		ast.NewLabel("last"),
		ast.NewVariable("", 1),
		ast.NewVariable("", 0),
		ast.NewLabel("end"),
	})
	assert.NoError(t, err)
	assert.Empty(t, assembly.Binary)
	assert.Equal(t, uint64(0xffff), assembly.Symbols["last"])
	assert.Equal(t, uint64(0x10000), assembly.Symbols["end"])
}

func TestCodec_ReservationInCodeKeepsFollowingByteAddress(t *testing.T) {
	t.Parallel()

	c := newReservationCodec(t, 0x200, 8)
	stream, err := c.ParseStream(t.Context(), "storage.asm", strings.NewReader(`
.segment "CODE"
.byte $a5
.res 3
after:
.byte $5a
`))
	assert.NoError(t, err)
	assembly, err := c.AssembleStream(t.Context(), stream)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xa5, 0, 0, 0, 0x5a}, assembly.Binary)
	assert.Equal(t, uint64(0x8004), assembly.Symbols["after"])
}

func TestCodec_TextReservationChecksRAMCapacity(t *testing.T) {
	t.Parallel()

	for _, size := range []int{8, 9} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			t.Parallel()

			c := newReservationCodec(t, 0x200, 8)
			text := fmt.Sprintf(".segment \"RAM\"\n.res %d\nend:\n", size)
			stream, err := c.ParseStream(t.Context(), "storage.asm", strings.NewReader(text))
			assert.NoError(t, err)
			assembly, err := c.AssembleStream(t.Context(), stream)
			if size > 8 {
				assert.Error(t, err)
				assert.Nil(t, assembly)
				return
			}
			assert.NoError(t, err)
			assert.Empty(t, assembly.Binary)
			assert.Equal(t, uint64(0x208), assembly.Symbols["end"])
		})
	}
}

func TestCodec_ReservationArchitectureBounds(t *testing.T) {
	t.Parallel()

	checkReservationArchitecture(t, "6502", asmcpu6502.New())
	checkReservationArchitecture(t, "65816", asmcpu65816.New())
	checkReservationArchitecture(t, "68000", asmcpu68000.New())
	checkReservationArchitecture(t, "CHIP-8", asmchip8.New())
	checkReservationArchitecture(t, "SM83", asmsm83.New())
	checkReservationArchitecture(t, "Z80", asmz80.New())
}

func TestCodec_ReservationDoesNotUseCodeAddressForOffsetCounter(t *testing.T) {
	t.Parallel()

	c := newReservationCodec(t, 0x200, 8)
	reservation := ast.NewVariable("scratch", 3)
	reservation.UseOffsetCounter = true
	assembly, err := c.Assemble(t.Context(), []ast.Node{
		ast.NewSegment("CODE"),
		reservation,
	})
	assert.ErrorContains(t, err, "offset-counter reservation")
	assert.Nil(t, assembly)
}

func checkReservationArchitecture[T any](t *testing.T, name string, configuration *config.Config[T]) {
	t.Helper()

	t.Run(name, func(t *testing.T) {
		end := uint64(1) << configuration.Arch.AddressWidth()
		ram := &config.Segment{
			Memory: config.Memory{
				Name:  "RAM",
				Start: end - 1,
				Size:  2,
			},
			SegmentName:  "RAM",
			SegmentStart: end - 1,
		}
		configuration.Segments = map[string]*config.Segment{"RAM": ram}
		configuration.SegmentsOrdered = []*config.Segment{ram}
		c, err := codec.New(configuration)
		assert.NoError(t, err)

		for _, size := range []int{1, 2} {
			assembly, assembleErr := c.Assemble(t.Context(), []ast.Node{
				ast.NewLabel("last"),
				ast.NewVariable("", size),
				ast.NewLabel("end"),
			})
			if size == 2 {
				assert.Error(t, assembleErr)
				assert.Nil(t, assembly)
				continue
			}
			assert.NoError(t, assembleErr)
			assert.Empty(t, assembly.Binary)
			assert.Equal(t, end-1, assembly.Symbols["last"])
			assert.Equal(t, end, assembly.Symbols["end"])
		}
	})
}

func newReservationCodec(t *testing.T, start, size uint64) *codec.Codec[*cpu6502.Instruction] {
	t.Helper()

	configuration := asmcpu6502.New()
	configuration.CompatibilityMode = config.CompatCa65
	ram := &config.Segment{
		Memory: config.Memory{
			Name:  "RAM",
			Start: start,
			Size:  size,
		},
		SegmentName:  "RAM",
		SegmentStart: start,
	}
	code := &config.Segment{
		Memory: config.Memory{
			Name:  "ROM",
			Start: 0x8000,
			Size:  16,
		},
		SegmentName:  "CODE",
		SegmentStart: 0x8000,
	}
	configuration.Segments = map[string]*config.Segment{
		"RAM":  ram,
		"CODE": code,
	}
	configuration.SegmentsOrdered = []*config.Segment{ram, code}
	c, err := codec.New(configuration)
	assert.NoError(t, err)
	return c
}
