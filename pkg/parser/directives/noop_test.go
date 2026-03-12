package directives

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retrogolib/assert"
)

func TestNoOp(t *testing.T) {
	parser := newMockParser([]token.Token{
		{Type: token.Dot, Value: "."},
		{Type: token.Identifier, Value: "list"},
		{Type: token.Identifier, Value: "on"},
		{Type: token.EOL},
	})

	node, err := NoOp(parser)
	assert.NoError(t, err)
	assert.Nil(t, node)
}

func TestBuildHandlers_Default(t *testing.T) {
	handlers := BuildHandlers(config.CompatDefault)
	// Should have base handlers
	_, ok := handlers["byte"]
	assert.True(t, ok)
	_, ok = handlers["org"]
	assert.True(t, ok)

	// Should NOT have x816-specific handlers
	_, ok = handlers["list"]
	assert.False(t, ok)
}

func TestBuildHandlers_X816(t *testing.T) {
	handlers := BuildHandlers(config.CompatX816)
	// Should have base handlers
	_, ok := handlers["byte"]
	assert.True(t, ok)

	// Should have x816-specific handlers
	_, ok = handlers["list"]
	assert.True(t, ok)
	_, ok = handlers["nolist"]
	assert.True(t, ok)
	_, ok = handlers["sym"]
	assert.True(t, ok)
}

func TestBuildHandlers_Ca65(t *testing.T) {
	handlers := BuildHandlers(config.CompatCa65)
	// Should have ca65-specific no-op directives
	_, ok := handlers["export"]
	assert.True(t, ok)
	_, ok = handlers["feature"]
	assert.True(t, ok)
}
