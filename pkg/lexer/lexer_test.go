package lexer

import (
	"strings"
	"testing"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retrogolib/assert"
)

//nolint:funlen // table-driven test with many cases
func TestLexer(t *testing.T) {
	tests := []struct {
		input    string
		expected []token.Token
	}{
		{`"test"`, []token.Token{{Type: token.Identifier, Value: `"test"`}}},
		{"", nil},
		{"rti", []token.Token{{Type: token.Identifier, Value: "rti"}}},
		{"label:", []token.Token{
			{Type: token.Identifier, Value: "label"},
			{Type: token.Colon},
		}},
		{"bcc  label", []token.Token{
			{Type: token.Identifier, Value: "bcc"},
			{Type: token.Identifier, Value: "label"},
		}},
		{" // comment", []token.Token{{Type: token.Comment, Value: "comment"}}},
		{" ;comment", []token.Token{{Type: token.Comment, Value: "comment"}}},
		{" var_a = 1", []token.Token{
			{Type: token.Identifier, Value: "var_a"},
			{Type: token.Assign},
			{Type: token.Number, Value: "1"},
		}},
		{".word func_a", []token.Token{
			{Type: token.Dot},
			{Type: token.Identifier, Value: "word"},
			{Type: token.Identifier, Value: "func_a"},
		}},
		{"lda (123),y", []token.Token{
			{Type: token.Identifier, Value: "lda"},
			{Type: token.LeftParentheses},
			{Type: token.Number, Value: "123"},
			{Type: token.RightParentheses},
			{Type: token.Comma},
			{Type: token.Identifier, Value: "y"},
		}},
		{"sta z:var_1", []token.Token{
			{Type: token.Identifier, Value: "sta"},
			{Type: token.Identifier, Value: "z"},
			{Type: token.Colon},
			{Type: token.Identifier, Value: "var_1"},
		}},
		{".byte $12, $00", []token.Token{
			{Type: token.Dot},
			{Type: token.Identifier, Value: "byte"},
			{Type: token.Number, Value: "$12"},
			{Type: token.Comma},
			{Type: token.Number, Value: "$00"},
		}},
		{"lda #$4C", []token.Token{
			{Type: token.Identifier, Value: "lda"},
			{Type: token.Number, Value: "#$4C"},
		}},
		{"lda #%10001000", []token.Token{
			{Type: token.Identifier, Value: "lda"},
			{Type: token.Number, Value: "#%10001000"},
		}},
		{"3c", []token.Token{{Type: token.Number, Value: "3c"}}},
	}

	cfg := Config{
		CommentPrefixes: []string{"//", ";"},
		DecimalPrefix:   '#',
	}

	for _, tt := range tests {
		l := New(cfg, strings.NewReader(tt.input))

		for _, expected := range tt.expected {
			tok, err := l.NextToken()
			assert.NoError(t, err)
			assert.Equal(t, expected.Type, tok.Type, "input: "+tt.input)
			assert.Equal(t, expected.Value, tok.Value, "input: "+tt.input)
		}

		tok, err := l.NextToken()
		assert.NoError(t, err)
		assert.Equal(t, token.EOF, tok.Type, tt.input)
	}
}

func TestLexerReadNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected token.Token
	}{
		{"0ABCDh/4", token.Token{Type: token.Number, Value: "0x0ABCD"}},
		{"0ABCDh", token.Token{Type: token.Number, Value: "0x0ABCD"}},
		{"0ABCDH", token.Token{Type: token.Number, Value: "0x0ABCD"}},
		{"$ABCD", token.Token{Type: token.Number, Value: "$ABCD"}},
		{"12345", token.Token{Type: token.Number, Value: "12345"}},
		{"%01010101", token.Token{Type: token.Number, Value: "%01010101"}},
		{"01010101b", token.Token{Type: token.Number, Value: "01010101b"}},
		{"#%10001000", token.Token{Type: token.Number, Value: "#%10001000"}},
		{"0x3c", token.Token{Type: token.Number, Value: "0x3c"}},
	}

	cfg := Config{
		CommentPrefixes: []string{"//", ";"},
		DecimalPrefix:   '#',
	}

	for _, tt := range tests {
		l := New(cfg, strings.NewReader(tt.input))
		tok, err := l.NextToken()
		assert.NoError(t, err)
		assert.Equal(t, tt.expected.Type, tok.Type, tt.input)
		assert.Equal(t, tt.expected.Value, tok.Value, tt.input)
	}
}
