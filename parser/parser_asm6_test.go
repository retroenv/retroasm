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
		expected func() []ast.Node
	}{
		{input: "INCBIN foo.bin, $200, $2000", expected: func() []ast.Node {
			return []ast.Node{ast.NewInclude("foo.bin", true, 0x200, 0x2000)}
		}},
		{input: "INCBIN foo.bin, $400", expected: func() []ast.Node {
			return []ast.Node{ast.NewInclude("foo.bin", true, 0x400, 0)}
		}},
		{input: "BIN \"../whatever.bin\"", expected: func() []ast.Node {
			return []ast.Node{ast.NewInclude("\"../whatever.bin\"", true, 0, 0)}
		}},
		{input: "BIN whatever.bin", expected: func() []ast.Node {
			return []ast.Node{ast.NewInclude("whatever.bin", true, 0, 0)}
		}},
		{input: "INCBIN whatever.bin", expected: func() []ast.Node {
			return []ast.Node{ast.NewInclude("whatever.bin", true, 0, 0)}
		}},
		{input: "INCSRC whatever.asm", expected: func() []ast.Node {
			return []ast.Node{ast.NewInclude("whatever.asm", false, 0, 0)}
		}},
		{input: "INCLUDE whatever.asm", expected: func() []ast.Node {
			return []ast.Node{ast.NewInclude("whatever.asm", false, 0, 0)}
		}},
		{input: "lda #12h", expected: func() []ast.Node {
			return []ast.Node{ast.NewInstruction("lda", ImmediateAddressing, ast.NewNumber(0x12), nil)}
		}},
	}

	architecture := arch.NewNES()

	for _, tt := range tests {
		parser := New(architecture, strings.NewReader(tt.input))
		assert.NoError(t, parser.Read())
		nodes, err := parser.TokensToAstNodes()
		assert.NoError(t, err, fmt.Sprintf("input: %s", tt.input))

		expectedNodes := tt.expected()
		for i, expected := range expectedNodes {
			assert.False(t, i >= len(nodes))

			node := nodes[i]
			assert.Equal(t, expected, node, fmt.Sprintf("input: %s", tt.input))
		}

		last := len(expectedNodes)
		for i := last; i < len(nodes); i++ {
			t.Errorf("unexpected node %v", nodes[i])
		}
	}
}
