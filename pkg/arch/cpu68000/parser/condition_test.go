package parser

import (
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

var conditionCodeTests = []struct {
	mnemonic string
	wantBase string
	wantCode uint16
	wantOK   bool
}{
	{"BEQ", "Bcc", 7, true},
	{"BNE", "Bcc", 6, true},
	{"BGT", "Bcc", 14, true},
	{"BLT", "Bcc", 13, true},
	{"BGE", "Bcc", 12, true},
	{"BLE", "Bcc", 15, true},
	{"BHI", "Bcc", 2, true},
	{"BLS", "Bcc", 3, true},
	{"BCC", "Bcc", 4, true},
	{"BCS", "Bcc", 5, true},
	{"BPL", "Bcc", 10, true},
	{"BMI", "Bcc", 11, true},
	{"BVC", "Bcc", 8, true},
	{"BVS", "Bcc", 9, true},
	// Case insensitive
	{"beq", "Bcc", 7, true},
	{"bne", "Bcc", 6, true},
	// DBcc variants
	{"DBEQ", "DBcc", 7, true},
	{"DBNE", "DBcc", 6, true},
	{"DBGT", "DBcc", 14, true},
	{"dbne", "DBcc", 6, true},
	// Scc variants
	{"SHI", "Scc", 2, true},
	{"SEQ", "Scc", 7, true},
	{"SNE", "Scc", 6, true},
	{"shi", "Scc", 2, true},
	// Not a condition code mnemonic
	{"MOVE", "MOVE", 0, false},
	{"NOP", "NOP", 0, false},
	{"ADD", "ADD", 0, false},
	// B prefix but not a valid condition
	{"BXYZ", "BXYZ", 0, false},
}

func TestParseConditionCode(t *testing.T) {
	for _, tt := range conditionCodeTests {
		t.Run(tt.mnemonic, func(t *testing.T) {
			base, code, ok := ParseConditionCode(tt.mnemonic)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantBase, base)
			assert.Equal(t, tt.wantCode, code)
		})
	}
}
