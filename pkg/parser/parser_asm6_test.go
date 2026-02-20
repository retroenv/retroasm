package parser

import (
	"strings"
	"testing"

	m6502Arch "github.com/retroenv/retroasm/pkg/arch/m6502"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/m6502"
	"github.com/retroenv/retrogolib/assert"
)

//nolint:funlen // table-driven test with many cases
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
			return []ast.Node{ast.NewInstruction("lda", int(m6502.ImmediateAddressing), ast.NewNumber(0x12), nil)}
		}},
	}

	cfg := m6502Arch.New()

	for _, tt := range tests {
		parser := New(cfg.Arch, strings.NewReader(tt.input))
		assert.NoError(t, parser.Read(t.Context()))
		nodes, err := parser.TokensToAstNodes()
		assert.NoError(t, err, "input: "+tt.input)

		expectedNodes := tt.expected()
		assert.Len(t, nodes, len(expectedNodes), "input: "+tt.input)
		for i, expected := range expectedNodes {
			assert.Equal(t, expected, nodes[i], "input: "+tt.input)
		}
	}
}
