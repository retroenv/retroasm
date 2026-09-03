package ast

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retrogolib/assert"
)

func TestParseSymbolReference(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tokens   []token.Token
		symbol   string
		addend   int64
		expected bool
	}{
		{name: "symbol", tokens: []token.Token{{Type: token.Identifier, Value: "entry"}}, symbol: "entry", expected: true},
		{name: "positive addend", tokens: []token.Token{{Type: token.Identifier, Value: "entry"}, {Type: token.Plus}, {Type: token.Number, Value: "2"}}, symbol: "entry", addend: 2, expected: true},
		{name: "negative addend", tokens: []token.Token{{Type: token.Identifier, Value: "entry"}, {Type: token.Minus}, {Type: token.Number, Value: "$10"}}, symbol: "entry", addend: -16, expected: true},
		{name: "constant first", tokens: []token.Token{{Type: token.Number, Value: "2"}, {Type: token.Plus}, {Type: token.Identifier, Value: "entry"}}, symbol: "entry", addend: 2, expected: true},
		{name: "number", tokens: []token.Token{{Type: token.Number, Value: "2"}}},
		{name: "nonlinear", tokens: []token.Token{{Type: token.Identifier, Value: "entry"}, {Type: token.Asterisk}, {Type: token.Number, Value: "2"}}},
		{name: "overflow", tokens: []token.Token{{Type: token.Identifier, Value: "entry"}, {Type: token.Plus}, {Type: token.Number, Value: "0xffffffffffffffff"}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			symbol, addend, ok := ParseSymbolReference(expression.New(test.tokens...))
			assert.Equal(t, test.expected, ok)
			assert.Equal(t, test.symbol, symbol)
			assert.Equal(t, test.addend, addend)
		})
	}
}
