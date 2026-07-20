package directives

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retrogolib/assert"
)

func TestNoOp(t *testing.T) {
	tests := []struct {
		name   string
		tokens []token.Token
	}{
		{
			name: "without arguments",
			tokens: []token.Token{
				{Type: token.Dot, Value: "."},
				{Type: token.Identifier, Value: "list"},
				{Type: token.EOL},
			},
		},
		{
			name: "with arguments",
			tokens: []token.Token{
				{Type: token.Dot, Value: "."},
				{Type: token.Identifier, Value: "list"},
				{Type: token.Identifier, Value: "on"},
				{Type: token.EOL},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parser := newMockParser(test.tokens)

			node, err := NoOp(parser)
			assert.NoError(t, err)
			assert.Nil(t, node)
			assert.True(t, parser.NextToken(0).Type.IsTerminator())
		})
	}
}

func TestBuildHandlers(t *testing.T) {
	modes := []config.CompatibilityMode{
		config.CompatDefault,
		config.CompatAsm6,
		config.CompatCa65,
		config.CompatNesasm,
		config.CompatX816,
	}

	for _, mode := range modes {
		t.Run(mode.String(), func(t *testing.T) {
			handlers := BuildHandlers(mode)

			_, byteFound := handlers["byte"]
			_, orgFound := handlers["org"]
			assert.True(t, byteFound)
			assert.True(t, orgFound)
		})
	}
}

func TestBuildHandlers_ReturnsIndependentMaps(t *testing.T) {
	first := BuildHandlers(config.CompatDefault)
	second := BuildHandlers(config.CompatDefault)

	delete(first, "byte")
	_, secondFound := second["byte"]

	assert.True(t, secondFound)
}
