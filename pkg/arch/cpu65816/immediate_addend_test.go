package cpu65816

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/retroenv/retroasm/pkg/assembler"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retrogolib/assert"
)

func TestImmediateSymbolAddendsAssembleExactBytes(t *testing.T) {
	for _, test := range []struct {
		name   string
		source string
		want   []byte
	}{
		{
			name:   "word positive and negative",
			source: "source = $1234\nREP #$30\nLDA #source+3\nLDX #source-3\nLDY #source+16\n",
			want:   []byte{0xc2, 0x30, 0xa9, 0x37, 0x12, 0xa2, 0x31, 0x12, 0xa0, 0x44, 0x12},
		},
		{
			name:   "byte positive and negative",
			source: "source = $80\nSEP #$30\nLDA #source+3\nLDX #source-3\nLDY #source+16\n",
			want:   []byte{0xe2, 0x30, 0xa9, 0x83, 0xa2, 0x7d, 0xa0, 0x90},
		},
		{
			name:   "compact and split numeric controls",
			source: "LDX #$12\nLDX # $34\nREP #$10\nLDX #$1234\nLDX # $5678\n",
			want:   []byte{0xa2, 0x12, 0xa2, 0x34, 0xc2, 0x10, 0xa2, 0x34, 0x12, 0xa2, 0x78, 0x56},
		},
		{
			name:   "PEA symbolic addend control",
			source: "source = $1234\nPEA source+3\nPEA source-3\n",
			want:   []byte{0xf4, 0x37, 0x12, 0xf4, 0x31, 0x12},
		},
		{
			name:   "separate local scopes",
			source: "first:\nREP #$10\nLDX #@target+1\n@target: .byte $AA\nsecond:\nLDX #@target-1\n@target: .byte $BB\n",
			want:   []byte{0xc2, 0x10, 0xa2, 0x06, 0x80, 0xaa, 0xa2, 0x08, 0x80, 0xbb},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			output, err := assembleImmediateAddend(t, test.source)
			assert.NoError(t, err)
			assert.Equal(t, test.want, output)
		})
	}
}

func TestImmediateSymbolAddendsRejectInvalidLinkedValues(t *testing.T) {
	for _, test := range []struct {
		name   string
		source string
	}{
		{name: "unknown", source: "REP #$10\nLDX #missing+3\n"},
		{name: "byte overflow", source: "source = $FF\nLDX #source+1\n"},
		{name: "word overflow", source: "source = $FFFF\nREP #$10\nLDX #source+1\n"},
		{name: "underflow", source: "source = 0\nREP #$10\nLDX #source-1\n"},
		{name: "missing positive addend", source: "source = $12\nLDX #source+\n"},
		{name: "missing negative addend", source: "source = $12\nLDX #source-\n"},
		{name: "invalid addend", source: "source = $12\nLDX #source+)\n"},
		{name: "integer overflow", source: "source = $12\nLDX #source+18446744073709551616\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := assembleImmediateAddend(t, test.source)
			assert.Error(t, err)
		})
	}
}

func assembleImmediateAddend(t *testing.T, source string) ([]byte, error) {
	t.Helper()
	configuration := New()
	configuration.CompatibilityMode = config.CompatCa65
	layout := `MEMORY { CODE: start = $8000, size = $100; }
SEGMENTS { CODE: load = CODE, type = ro; }`
	if err := configuration.ReadCa65Config(strings.NewReader(layout)); err != nil {
		return nil, fmt.Errorf("reading immediate addend layout: %w", err)
	}
	var output bytes.Buffer
	asm := assembler.New(configuration, &output)
	if err := asm.Process(t.Context(), strings.NewReader(".segment \"CODE\"\n"+source)); err != nil {
		return nil, fmt.Errorf("assembling immediate addend: %w", err)
	}
	return output.Bytes(), nil
}
