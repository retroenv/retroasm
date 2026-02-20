package token

import (
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

func TestTokenToToken(t *testing.T) {
	for r, typ := range toToken {
		found, ok := toString[typ]
		assert.True(t, ok, "token type %v should exist in toString map", typ)
		assert.Equal(t, rune(found[0]), r)
	}
}
