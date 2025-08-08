package directives

import (
	"testing"

	"github.com/retroenv/retroasm/lexer/token"
	"github.com/retroenv/retroasm/parser/ast"
	"github.com/retroenv/retrogolib/assert"
)

// mockParser provides a simple parser implementation for testing directives.
type mockParser struct {
	tokens   []token.Token
	position int
}

func newMockParser(tokens []token.Token) *mockParser {
	return &mockParser{
		tokens:   tokens,
		position: 0,
	}
}

func (p *mockParser) NextToken(offset int) token.Token {
	pos := p.position + offset
	if pos >= len(p.tokens) {
		return token.Token{Type: token.EOF}
	}
	return p.tokens[pos]
}

func (p *mockParser) AdvanceReadPosition(offset int) {
	p.position += offset
}

func (p *mockParser) AddressWidth() int {
	return 16
}

func TestSetCPU(t *testing.T) {
	parser := newMockParser([]token.Token{
		{Type: token.Dot, Value: "."},
		{Type: token.Identifier, Value: "setcpu"},
		{Type: token.Identifier, Value: "6502"},
	})

	node, err := SetCPU(parser)
	assert.NoError(t, err)
	assert.Nil(t, node)                 // SetCPU returns nil node
	assert.Equal(t, 2, parser.position) // Should advance position by 2
}

// Test integration with mock parser.
func TestDirectiveIntegration(t *testing.T) {
	t.Run("data_directive", func(t *testing.T) {
		// Create a simple mock for testing data directive
		parser := newMockParser([]token.Token{
			{Type: token.Dot, Value: "."},
			{Type: token.Identifier, Value: "byte"},
			{Type: token.Number, Value: "$10"},
			{Type: token.EOL},
		})

		node, err := Data(parser)
		assert.NoError(t, err)
		assert.NotNil(t, node)

		// Handle both value and pointer types
		if data, ok := node.(*ast.Data); ok {
			assert.Equal(t, ast.DataType, data.Type)
		} else if data, ok := node.(ast.Data); ok {
			assert.Equal(t, ast.DataType, data.Type)
		} else {
			t.Fatalf("Expected Data node, got %T", node)
		}
	})

	t.Run("base_directive", func(t *testing.T) {
		parser := newMockParser([]token.Token{
			{Type: token.Dot, Value: "."},
			{Type: token.Identifier, Value: "org"},
			{Type: token.Number, Value: "$8000"},
			{Type: token.EOL},
		})

		node, err := Base(parser)
		assert.NoError(t, err)
		assert.NotNil(t, node)

		// Handle both value and pointer types
		if base, ok := node.(*ast.Base); ok {
			assert.NotNil(t, base.Address)
		} else if base, ok := node.(ast.Base); ok {
			assert.NotNil(t, base.Address)
		} else {
			t.Fatalf("Expected Base node, got %T", node)
		}
	})
}

// Benchmark critical directive parsing.
func BenchmarkDirectiveParsing(b *testing.B) {
	parser := newMockParser([]token.Token{
		{Type: token.Dot, Value: "."},
		{Type: token.Identifier, Value: "byte"},
		{Type: token.Number, Value: "$FF"},
		{Type: token.EOL},
	})

	b.Run("data_directive", func(b *testing.B) {
		for range b.N {
			p := &mockParser{tokens: parser.tokens, position: 0}
			_, _ = Data(p)
		}
	})

	orgParser := newMockParser([]token.Token{
		{Type: token.Dot, Value: "."},
		{Type: token.Identifier, Value: "org"},
		{Type: token.Number, Value: "$8000"},
		{Type: token.EOL},
	})

	b.Run("base_directive", func(b *testing.B) {
		for range b.N {
			p := &mockParser{tokens: orgParser.tokens, position: 0}
			_, _ = Base(p)
		}
	})
}
