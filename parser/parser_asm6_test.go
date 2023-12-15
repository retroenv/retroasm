package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/retroenv/assembler/arch"
	"github.com/retroenv/assembler/parser/ast"
	. "github.com/retroenv/retrogolib/addressing"
	"github.com/retroenv/retrogolib/assert"
)

// nolint: funlen
func TestParserAsm6(t *testing.T) {
	tests := []struct {
		input    string
		expected []ast.Node
	}{
		{input: "INCBIN foo.bin, $200, $2000", expected: []ast.Node{
			&ast.Include{
				Name:   "foo.bin",
				Binary: true,
				Start:  0x200,
				Size:   0x2000,
			},
		}},
		{input: "INCBIN foo.bin, $400", expected: []ast.Node{
			&ast.Include{
				Name:   "foo.bin",
				Binary: true,
				Start:  0x400,
			},
		}},
		{input: "BIN \"../whatever.bin\"", expected: []ast.Node{
			&ast.Include{
				Name:   "\"../whatever.bin\"",
				Binary: true,
			},
		}},
		{input: "BIN whatever.bin", expected: []ast.Node{
			&ast.Include{
				Name:   "whatever.bin",
				Binary: true,
			},
		}},
		{input: "INCBIN whatever.bin", expected: []ast.Node{
			&ast.Include{
				Name:   "whatever.bin",
				Binary: true,
			},
		}},
		{input: "INCSRC whatever.asm", expected: []ast.Node{
			&ast.Include{
				Name: "whatever.asm",
			},
		}},
		{input: "INCLUDE whatever.asm", expected: []ast.Node{
			&ast.Include{
				Name: "whatever.asm",
			},
		}},
		{input: "lda #12h", expected: []ast.Node{
			&ast.Instruction{
				Name:       "lda",
				Addressing: ImmediateAddressing,
				Argument:   ast.Number{Value: 0x12},
			},
		}},
	}

	architecture := arch.NewNES()

	for _, tt := range tests {
		parser := New(architecture, strings.NewReader(tt.input))
		nodes, err := parser.Read()
		assert.NoError(t, err, fmt.Sprintf("input: %s", tt.input))

		for i, expected := range tt.expected {
			assert.False(t, i >= len(nodes))

			node := nodes[i]
			assert.Equal(t, expected, node, fmt.Sprintf("input: %s", tt.input))
		}

		last := len(tt.expected)
		for i := last; i < len(nodes); i++ {
			t.Errorf("unexpected node %v", nodes[i])
		}
	}
}
