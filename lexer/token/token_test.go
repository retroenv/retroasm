package token

import (
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

func TestTokenToToken(t *testing.T) {
	for r, typ := range toToken {
		found, ok := toString[typ]
		assert.True(t, ok)
		assert.Equal(t, found[0], r)
	}
}
