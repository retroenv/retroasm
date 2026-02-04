package assembler

import (
	"testing"
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
