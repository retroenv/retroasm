package parser

import (
	"context"
	"strings"
	"testing"

	m6502Arch "github.com/retroenv/retroasm/arch/m6502"
	"github.com/retroenv/retroasm/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/m6502"
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
			return []ast.Node{ast.NewInstruction("asl", int(m6502.AbsoluteAddressing), l, nil)}
		}},
		{"asl a:1", func() []ast.Node {
			return []ast.Node{ast.NewInstruction("asl", int(m6502.AbsoluteAddressing), ast.NewNumber(1), nil)}
		}},
		{"asl", func() []ast.Node {
			return []ast.Node{ast.NewInstruction("asl", int(m6502.AccumulatorAddressing), nil, nil)}
		}},
		{"asl a", func() []ast.Node {
			return []ast.Node{ast.NewInstruction("asl", int(m6502.AccumulatorAddressing), nil, nil)}
		}},
		{"asl\na:", func() []ast.Node {
			l := ast.NewLabel("a")
			return []ast.Node{
				ast.NewInstruction("asl", int(m6502.AccumulatorAddressing), nil, nil),
				l,
			}
		}},
	}

	cfg := m6502Arch.New()

	for _, tt := range tests {
		parser := New(cfg.Arch, strings.NewReader(tt.input))
		assert.NoError(t, parser.Read(context.Background()))
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
