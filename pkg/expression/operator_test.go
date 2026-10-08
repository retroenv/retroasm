package expression

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retrogolib/assert"
)

func TestEvaluateOperatorIntInt_ModuloByZero(t *testing.T) {
	_, err := evaluateOperatorIntInt(token.Percent, 6, 0)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errDivisionByZero)
}

func TestEvaluateOperatorByteByte_UnsupportedOperator(t *testing.T) {
	a := []byte{1, 2}
	b := []byte{3, 4}
	_, err := evaluateOperatorByteByte(token.Equals, a, b)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "[]byte and []byte")
}

func TestEvaluateOperatorByteByte_OperandLengths(t *testing.T) {
	tests := []struct {
		name string
		a, b []byte
		want []byte
	}{
		{
			name: "repeat right operand",
			a:    []byte{1, 2, 3, 4, 5},
			b:    []byte{10, 20},
			want: []byte{11, 22, 13, 24, 15},
		},
		{
			name: "longer right operand",
			a:    []byte{1, 2},
			b:    []byte{10, 20, 30},
			want: []byte{11, 22},
		},
		{
			name: "empty right operand",
			a:    []byte{1, 2},
			want: []byte{1, 2},
		},
		{
			name: "empty left operand",
			b:    []byte{10, 20},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := evaluateOperatorByteByte(token.Plus, test.a, test.b)
			assert.NoError(t, err)
			assert.Equal(t, test.want, result)
		})
	}
}

func TestEvaluateOperatorByteInt_DivisionByZero(t *testing.T) {
	a := []byte{10, 20}
	_, err := evaluateOperatorByteInt(token.Slash, a, 0)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errDivisionByZero)
}
