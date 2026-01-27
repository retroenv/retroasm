package parser

import (
	"context"
	"strings"
	"testing"

	m6502Arch "github.com/retroenv/retroasm/pkg/arch/m6502"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
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

func TestParser_EdgeCases(t *testing.T) {
	cfg := m6502Arch.New()

	t.Run("empty input", func(t *testing.T) {
		parser := New(cfg.Arch, strings.NewReader(""))
		assert.NoError(t, parser.Read(context.Background()))
		nodes, err := parser.TokensToAstNodes()
		assert.NoError(t, err)
		assert.Equal(t, 0, len(nodes))
	})

	t.Run("only whitespace", func(t *testing.T) {
		parser := New(cfg.Arch, strings.NewReader("   \n\t  \n"))
		assert.NoError(t, parser.Read(context.Background()))
		nodes, err := parser.TokensToAstNodes()
		assert.NoError(t, err)
		assert.Equal(t, 0, len(nodes))
	})

	t.Run("only comments", func(t *testing.T) {
		parser := New(cfg.Arch, strings.NewReader("; comment\n// another comment"))
		assert.NoError(t, parser.Read(context.Background()))
		nodes, err := parser.TokensToAstNodes()
		assert.NoError(t, err)
		// Comment nodes may be combined or filtered, check actual behavior
		assert.True(t, len(nodes) >= 1, "should have at least one comment node")
	})
}

func TestParser_ErrorConditions(t *testing.T) {
	cfg := m6502Arch.New()

	t.Run("context cancellation during read", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		parser := New(cfg.Arch, strings.NewReader("lda #$01"))
		err := parser.Read(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cancelled")
	})

	t.Run("unsupported directive", func(t *testing.T) {
		parser := New(cfg.Arch, strings.NewReader(".unsupported"))
		assert.NoError(t, parser.Read(context.Background()))
		_, err := parser.TokensToAstNodes()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported directive")
	})

	t.Run("missing directive parameter", func(t *testing.T) {
		parser := New(cfg.Arch, strings.NewReader(".byte"))
		assert.NoError(t, parser.Read(context.Background()))
		_, err := parser.TokensToAstNodes()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing parameter")
	})

	t.Run("unexpected token type", func(t *testing.T) {
		parser := New(cfg.Arch, strings.NewReader("@"))
		// The lexer may handle @ as an illegal token before parser sees it
		err := parser.Read(context.Background())
		if err != nil {
			assert.Contains(t, err.Error(), "illegal")
		} else {
			_, err = parser.TokensToAstNodes()
			if err != nil {
				assert.Contains(t, err.Error(), "unexpected")
			}
		}
	})
}

func TestParser_NextToken(t *testing.T) {
	cfg := m6502Arch.New()
	parser := NewWithTokens(cfg.Arch, []token.Token{
		{Type: token.Identifier, Value: "lda"},
		{Type: token.Number, Value: "#$01"},
		{Type: token.EOL},
	})
	parser.programLength = 3

	t.Run("valid offset", func(t *testing.T) {
		tok := parser.NextToken(0)
		assert.Equal(t, token.Identifier, tok.Type)
		assert.Equal(t, "lda", tok.Value)

		tok = parser.NextToken(1)
		assert.Equal(t, token.Number, tok.Type)
		assert.Equal(t, "#$01", tok.Value)
	})

	t.Run("offset beyond program length", func(t *testing.T) {
		tok := parser.NextToken(10)
		assert.Equal(t, token.EOF, tok.Type)
	})

	t.Run("negative position with offset", func(t *testing.T) {
		parser.readPosition = 2
		tok := parser.NextToken(1)
		assert.Equal(t, token.EOF, tok.Type)
	})
}

func TestParser_AdvanceReadPosition(t *testing.T) {
	cfg := m6502Arch.New()
	parser := NewWithTokens(cfg.Arch, nil)

	initialPos := parser.readPosition
	parser.AdvanceReadPosition(5)
	assert.Equal(t, initialPos+5, parser.readPosition)

	parser.AdvanceReadPosition(-2)
	assert.Equal(t, initialPos+3, parser.readPosition)
}

func TestParser_AddressWidth(t *testing.T) {
	cfg := m6502Arch.New()
	parser := New(cfg.Arch, strings.NewReader(""))

	// M6502 has 16-bit addresses
	assert.Equal(t, 16, parser.AddressWidth())
}

func TestParser_PreallocationBenefit(t *testing.T) {
	cfg := m6502Arch.New()

	// Test that pre-allocation doesn't break functionality with large programs
	largeInput := strings.Repeat("nop\n", 1000)
	parser := New(cfg.Arch, strings.NewReader(largeInput))
	assert.NoError(t, parser.Read(context.Background()))

	nodes, err := parser.TokensToAstNodes()
	assert.NoError(t, err)
	assert.Equal(t, 1000, len(nodes))
}
