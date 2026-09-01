package parser

import (
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

var parseRegisterListTests = []struct {
	input   string
	want    uint16
	wantErr bool
}{
	// Single data register
	{"D0", 0x0001, false},
	{"D7", 0x0080, false},
	// Single address register
	{"A0", 0x0100, false},
	{"A7", 0x8000, false},
	// Data register range
	{"D0-D3", 0x000F, false},
	{"D4-D7", 0x00F0, false},
	// Address register range
	{"A0-A2", 0x0700, false},
	// Mixed with slash separator
	{"D0-D3/A0-A2", 0x070F, false},
	{"D0/D1/D2", 0x0007, false},
	{"D0/A0", 0x0101, false},
	// All data registers
	{"D0-D7", 0x00FF, false},
	// All address registers
	{"A0-A7", 0xFF00, false},
	// Error: invalid register name
	{"X0", 0, true},
	// Error: reversed range
	{"D3-D1", 0, true},
}

func TestParseRegisterList(t *testing.T) {
	for _, tt := range parseRegisterListTests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseRegisterList(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
