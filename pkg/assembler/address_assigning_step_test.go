package assembler

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/assert"
)

func TestParseReferenceOffset(t *testing.T) {
	tests := []struct {
		input      string
		wantName   string
		wantOffset int64
	}{
		{"symbol", "symbol", 0},
		{"tileData+8", "tileData", 8},
		{"base-3", "base", -3},
		{"my_var+128", "my_var", 128},
		{"nooffset", "nooffset", 0},
		{"a+0", "a", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			name, offset := parseReferenceOffset(tt.input)
			if name != tt.wantName {
				t.Errorf("parseReferenceOffset(%q) name = %q, want %q", tt.input, name, tt.wantName)
			}
			if offset != tt.wantOffset {
				t.Errorf("parseReferenceOffset(%q) offset = %d, want %d", tt.input, offset, tt.wantOffset)
			}
		})
	}
}

func TestAssignVariableAddress(t *testing.T) {
	aa := addressAssign[any]{
		programCounter: 0x200,
	}
	v := &variable{
		v: ast.NewVariable("test", 4),
	}

	result := assignVariableAddress(aa, v)
	assert.Equal(t, uint64(0x204), result)
	assert.Equal(t, uint64(0x200), v.address)
}
