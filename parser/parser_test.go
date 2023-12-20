package parser

import (
	"strings"
	"testing"

	"github.com/retroenv/assembler/arch"
	"github.com/retroenv/assembler/parser/ast"
	. "github.com/retroenv/retrogolib/addressing"
	"github.com/retroenv/retrogolib/assert"
)

// nolint: funlen
func TestParser_Instruction(t *testing.T) {
	tests := []struct {
		input    string
		expected []ast.Node
	}{
		{"asl a:var1", []ast.Node{
			&ast.Instruction{
				Name:       "asl",
				Addressing: AbsoluteAddressing,
				Argument:   &ast.Label{Name: "var1"},
			},
		}},
		{"asl a:1", []ast.Node{
			&ast.Instruction{
				Name:       "asl",
				Addressing: AbsoluteAddressing,
				Argument:   ast.Number{Value: 1},
			},
		}},
		{"asl", []ast.Node{
			&ast.Instruction{
				Name:       "asl",
				Addressing: AccumulatorAddressing,
			},
		}},
		{"asl a", []ast.Node{
			&ast.Instruction{
				Name:       "asl",
				Addressing: AccumulatorAddressing,
			},
		}},
		{"asl\na:", []ast.Node{
			&ast.Instruction{
				Name:       "asl",
				Addressing: AccumulatorAddressing,
			},
			&ast.Label{
				Name: "a",
			},
		}},
	}

	architecture := arch.NewNES()

	for _, tt := range tests {
		parser := New(architecture, strings.NewReader(tt.input))
		assert.NoError(t, parser.Read())
		nodes, err := parser.TokensToAstNodes()
		assert.NoError(t, err)

		for i, expected := range tt.expected {
			assert.False(t, i >= len(nodes))

			node := nodes[i]
			assert.Equal(t, expected, node)
		}

		last := len(tt.expected)
		for i := last; i < len(nodes); i++ {
			t.Errorf("unexpected node %v", nodes[i])
		}
	}
}
