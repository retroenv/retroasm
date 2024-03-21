package expression

import (
	"fmt"
	"strings"
	"testing"

	"github.com/retroenv/retroasm/lexer"
	"github.com/retroenv/retroasm/lexer/token"
	"github.com/retroenv/retroasm/scope"
	"github.com/retroenv/retrogolib/assert"
)

func TestExpression(t *testing.T) {
	tests := []struct {
		input       string
		expected    any
		expectedErr bool
	}{
		{input: "2\n4", expected: []byte{2, 4}},
		{input: "2-4", expected: -2},
		{input: "2^3", expected: 8},
		{input: "6/2", expected: 3},
		{input: "6%4", expected: 2},
		{input: "(1+2)*2", expected: 6},
		{input: "((1+2)*2)+1", expected: 7},
		{input: "1+2*2", expected: 5},
		{input: "1+2-3+4", expected: 4},
		{input: "(1+2)*", expectedErr: true},
		{input: "(1+2", expectedErr: true},
		{input: "1+2)", expectedErr: true},
		{input: "1+a", expectedErr: true},
		{input: "1/0", expectedErr: true},
	}

	lexerCfg := lexer.Config{
		CommentPrefixes: []string{";"},
		DecimalPrefix:   '#',
	}

	for _, tt := range tests {
		lex := lexer.New(lexerCfg, strings.NewReader(tt.input))
		result, err := runEvaluation(t, lex)

		if tt.expectedErr {
			assert.True(t, err != nil, fmt.Sprintf("input: %s", tt.input))
		} else {
			assert.NoError(t, err, fmt.Sprintf("input: %s", tt.input))
		}

		if tt.expected == nil {
			tt.expected = 0
		}
		assert.Equal(t, tt.expected, result, fmt.Sprintf("input: %s", tt.input))
	}
}

func runEvaluation(t *testing.T, lex *lexer.Lexer) (any, error) {
	t.Helper()

	sc := scope.New(nil)
	var e Expression

	for {
		tok, err := lex.NextToken()
		assert.NoError(t, err)
		if tok.Type == token.Illegal {
			t.Fail()
		}
		if tok.Type == token.EOF {
			break
		}
		if tok.Type.IsTerminator() {
			continue
		}
		e.nodes = append(e.nodes, tok)
	}

	return e.Evaluate(sc, 1)
}
