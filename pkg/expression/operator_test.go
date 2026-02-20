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

func TestEvaluateOperatorByteInt_DivisionByZero(t *testing.T) {
	a := []byte{10, 20}
	_, err := evaluateOperatorByteInt(token.Slash, a, 0)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errDivisionByZero)
}
