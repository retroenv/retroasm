package parser

import (
	"strings"
	"testing"

	"github.com/retroenv/retroasm/arch"
	"github.com/retroenv/retroasm/parser/ast"
	. "github.com/retroenv/retrogolib/addressing"
	"github.com/retroenv/retrogolib/assert"
)

// nolint: funlen
func TestParser_Instruction(t *testing.T) {
	tests := []struct {
		input    string
		expected func() []ast.Node
	}{
		{"asl a:var1", func() []ast.Node {
			l := ast.NewLabel("var1")
			return []ast.Node{ast.NewInstruction("asl", AbsoluteAddressing, l, nil)}
		}},
		{"asl a:1", func() []ast.Node {
			return []ast.Node{ast.NewInstruction("asl", AbsoluteAddressing, ast.NewNumber(1), nil)}
		}},
		{"asl", func() []ast.Node {
			return []ast.Node{ast.NewInstruction("asl", AccumulatorAddressing, nil, nil)}
		}},
		{"asl a", func() []ast.Node {
			return []ast.Node{ast.NewInstruction("asl", AccumulatorAddressing, nil, nil)}
		}},
		{"asl\na:", func() []ast.Node {
			l := ast.NewLabel("a")
			return []ast.Node{
				ast.NewInstruction("asl", AccumulatorAddressing, nil, nil),
				l,
			}
		}},
	}

	architecture := arch.NewNES()

	for _, tt := range tests {
		parser := New(architecture, strings.NewReader(tt.input))
		assert.NoError(t, parser.Read())
		nodes, err := parser.TokensToAstNodes()
		assert.NoError(t, err)

		expectedNodes := tt.expected()
		for i, expected := range expectedNodes {
			assert.False(t, i >= len(nodes))

			node := nodes[i]
			assert.Equal(t, expected, node)
		}

		last := len(expectedNodes)
		for i := last; i < len(nodes); i++ {
			t.Errorf("unexpected node %v", nodes[i])
		}
	}
}
