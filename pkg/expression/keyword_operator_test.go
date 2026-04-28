package expression

import (
	"strings"
	"testing"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/scope"
	"github.com/retroenv/retrogolib/assert"
)

var keywordOperatorTests = []struct {
	name     string
	tokens   []token.Token
	expected int64
}{
	{
		name:     keywordOperatorShiftLeft,
		tokens:   []token.Token{{Type: token.Number, Value: "1"}, {Type: token.Identifier, Value: keywordOperatorShiftLeft}, {Type: token.Number, Value: "4"}},
		expected: 16,
	},
	{
		name:     keywordOperatorShiftRight,
		tokens:   []token.Token{{Type: token.Number, Value: "16"}, {Type: token.Identifier, Value: keywordOperatorShiftRight}, {Type: token.Number, Value: "2"}},
		expected: 4,
	},
	{
		name:     keywordOperatorAnd,
		tokens:   []token.Token{{Type: token.Number, Value: "$FF"}, {Type: token.Identifier, Value: keywordOperatorAnd}, {Type: token.Number, Value: "$0F"}},
		expected: 0x0F,
	},
	{
		name:     keywordOperatorOr,
		tokens:   []token.Token{{Type: token.Number, Value: "$F0"}, {Type: token.Identifier, Value: keywordOperatorOr}, {Type: token.Number, Value: "$0F"}},
		expected: 0xFF,
	},
	{
		name:     keywordOperatorXor,
		tokens:   []token.Token{{Type: token.Number, Value: "$FF"}, {Type: token.Identifier, Value: keywordOperatorXor}, {Type: token.Number, Value: "$0F"}},
		expected: 0xF0,
	},
	{
		name:     "case insensitive shl",
		tokens:   []token.Token{{Type: token.Number, Value: "1"}, {Type: token.Identifier, Value: strings.ToLower(keywordOperatorShiftLeft)}, {Type: token.Number, Value: "3"}},
		expected: 8,
	},
}

func TestKeywordOperators(t *testing.T) {
	for _, tt := range keywordOperatorTests {
		t.Run(tt.name, func(t *testing.T) {
			sc := scope.New(nil)
			expr := New(tt.tokens...)
			result, err := expr.Evaluate(sc, 1)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBitwiseOperatorTokenTypes(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []token.Token
		expected int64
	}{
		{
			name:     "pipe as bitwise OR",
			tokens:   []token.Token{{Type: token.Number, Value: "$F0"}, {Type: token.Pipe}, {Type: token.Number, Value: "$0F"}},
			expected: 0xFF,
		},
		{
			name:     "ampersand as bitwise AND",
			tokens:   []token.Token{{Type: token.Number, Value: "$FF"}, {Type: token.Ampersand}, {Type: token.Number, Value: "$0F"}},
			expected: 0x0F,
		},
		{
			name:     "shift left token",
			tokens:   []token.Token{{Type: token.Number, Value: "1"}, {Type: token.ShiftLeft}, {Type: token.Number, Value: "8"}},
			expected: 256,
		},
		{
			name:     "shift right token",
			tokens:   []token.Token{{Type: token.Number, Value: "256"}, {Type: token.ShiftRight}, {Type: token.Number, Value: "4"}},
			expected: 16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := scope.New(nil)
			expr := New(tt.tokens...)
			result, err := expr.Evaluate(sc, 1)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
