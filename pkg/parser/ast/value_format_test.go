package ast

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retrogolib/assert"
)

func TestFormatValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   Node
		options ValueFormatOptions
		want    string
	}{
		{name: "hex number", value: NewNumber(0x2a), want: "0x2A"},
		{
			name: "padded number", value: NewNumber(5),
			options: ValueFormatOptions{MinimumHexDigits: 4}, want: "0x0005",
		},
		{
			name: "decimal number", value: NewNumber(17),
			options: ValueFormatOptions{Decimal: true}, want: "17",
		},
		{name: "label", value: NewLabel("target"), want: "target"},
		{name: "identifier", value: NewIdentifier("constant"), want: "constant"},
		{
			name: "expression",
			value: NewExpression(
				token.Token{Type: token.Identifier, Value: "base"},
				token.Token{Type: token.Plus, Value: "+"},
				token.Token{Type: token.Number, Value: "$10"},
			),
			want: "base+0x10",
		},
		{
			name: "modulo expression",
			value: NewExpression(
				token.Token{Type: token.Identifier, Value: "base"},
				token.Token{Type: token.Percent, Value: "%"},
				token.Token{Type: token.Number, Value: "16"},
			),
			want: "base % 0x10",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			formatted, err := FormatValue(test.value, test.options)

			assert.NoError(t, err)
			assert.Equal(t, test.want, formatted)
		})
	}
}

func TestFormatValue_RejectsUnsupportedNode(t *testing.T) {
	t.Parallel()

	_, err := FormatValue(nil, ValueFormatOptions{})

	assert.Error(t, err)
}
