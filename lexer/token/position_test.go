package token

import (
	"testing"

	"github.com/retroenv/retrogolib/assert"
)

func TestPosition(t *testing.T) {
	p := Position{
		Line:   1,
		Column: 1,
	}

	p.NextLine()
	assert.Equal(t, 2, p.Line)
	assert.Equal(t, 0, p.Column)
}
