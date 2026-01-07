package expression

import (
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
		expectError bool
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
		{input: "(1+2)*", expectError: true},
		{input: "(1+2", expectError: true},
		{input: "1+2)", expectError: true},
		{input: "1+a", expectError: true},
		{input: "1/0", expectError: true},
	}

	lexerCfg := lexer.Config{
		CommentPrefixes: []string{";"},
		DecimalPrefix:   '#',
	}

	for _, tt := range tests {
		lex := lexer.New(lexerCfg, strings.NewReader(tt.input))
		result, err := runEvaluation(t, lex)

		if tt.expectError {
			assert.NotNil(t, err, "expected error for input: "+tt.input)
		} else {
			assert.NoError(t, err, "input: "+tt.input)
		}

		if tt.expected == nil {
			tt.expected = 0
		}
		assert.Equal(t, tt.expected, result, "input: "+tt.input)
	}
}

func TestExpression_StructuredErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedErr error
	}{
		{
			name:        "division by zero",
			input:       "5/0",
			expectedErr: errDivisionByZero,
		},
		{
			name:        "missing left parenthesis",
			input:       "1+2)",
			expectedErr: errMismatchedParenthesis,
		},
		{
			name:        "missing right parenthesis",
			input:       "(1+2",
			expectedErr: errMismatchedParenthesis,
		},
	}

	lexerCfg := lexer.Config{
		CommentPrefixes: []string{";"},
		DecimalPrefix:   '#',
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lex := lexer.New(lexerCfg, strings.NewReader(tt.input))
			_, err := runEvaluation(t, lex)

			assert.Error(t, err)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestExpression_InputValidation(t *testing.T) {
	sc := scope.New(nil)
	expr := New(token.Token{Type: token.Number, Value: "42"})

	// Test invalid data width (negative values)
	_, err := expr.Evaluate(sc, -1)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid data width")

	// Test valid data widths (0 is allowed for binary data, 1+ for numeric data)
	_, err = expr.Evaluate(sc, 0)
	assert.NoError(t, err)

	_, err = expr.Evaluate(sc, 1)
	assert.NoError(t, err)
}

func TestExpression_IntValue(t *testing.T) {
	expr := &Expression{}

	// Test not evaluated
	_, err := expr.IntValue()
	assert.ErrorIs(t, err, errExpressionNotEvaluated)

	// Test wrong type
	expr.value = "not an int"
	expr.evaluated = true
	_, err = expr.IntValue()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "unexpected expression value type")

	// Test correct type
	expr.value = int64(42)
	val, err := expr.IntValue()
	assert.NoError(t, err)
	assert.Equal(t, int64(42), val)
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
