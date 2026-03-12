package parser

import (
	"testing"

	z80profile "github.com/retroenv/retroasm/pkg/arch/z80/profile"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
	"github.com/retroenv/retrogolib/assert"
)

func TestParseIdentifierWithProfile(t *testing.T) {
	t.Run("default profile allows sll", func(t *testing.T) {
		parser := newMockParser(
			token.Token{Type: token.Identifier, Value: "sll"},
			token.Token{Type: token.Identifier, Value: "a"},
			token.Token{Type: token.EOL},
		)

		_, err := ParseIdentifierWithProfile(
			parser,
			cpuz80.SllName,
			[]*cpuz80.Instruction{cpuz80.CBSll},
			z80profile.Default,
		)
		assert.NoError(t, err)
	})

	t.Run("strict profile rejects sll", func(t *testing.T) {
		parser := newMockParser(
			token.Token{Type: token.Identifier, Value: "sll"},
			token.Token{Type: token.Identifier, Value: "a"},
			token.Token{Type: token.EOL},
		)

		_, err := ParseIdentifierWithProfile(
			parser,
			cpuz80.SllName,
			[]*cpuz80.Instruction{cpuz80.CBSll},
			z80profile.StrictDocumented,
		)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "strict-documented")
		assert.ErrorContains(t, err, "undocumented")
	})

	t.Run("gameboy profile rejects in a,(n)", func(t *testing.T) {
		parser := newMockParser(
			token.Token{Type: token.Identifier, Value: "in"},
			token.Token{Type: token.Identifier, Value: "a"},
			token.Token{Type: token.Comma},
			token.Token{Type: token.LeftParentheses},
			token.Token{Type: token.Number, Value: "$12"},
			token.Token{Type: token.RightParentheses},
			token.Token{Type: token.EOL},
		)

		_, err := ParseIdentifierWithProfile(
			parser,
			cpuz80.InName,
			[]*cpuz80.Instruction{cpuz80.InPort},
			z80profile.GameBoySubset,
		)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "gameboy-z80-subset")
	})
}
