package assembler

import (
	"bytes"
	"math"
	"strconv"
	"testing"

	asmchip8 "github.com/retroenv/retroasm/pkg/arch/chip8"
	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/assert"
)

func TestAssemblerRejectsNegativeProgramCounterFill(t *testing.T) {
	prefix := ast.NewData(ast.DataType, 1)
	prefix.Fill = true
	prefix.Size = expression.New(token.Token{
		Type:  token.Number,
		Value: "48",
	})
	fill := ast.NewData(ast.DataType, 1)
	fill.Fill = true
	fill.Size = expression.New(token.Token{
		Type:  token.Number,
		Value: "544",
	}, token.Token{
		Type: token.Minus,
	}, token.Token{
		Type:  token.Number,
		Value: expression.ProgramCounterReference,
	})
	var output bytes.Buffer
	assembly := New(asmchip8.New(), &output)
	err := assembly.ProcessAST(t.Context(), []ast.Node{prefix, fill})
	assert.ErrorContains(t, err, "negative")
	assert.Empty(t, output.Bytes())
}

func TestGenerateDataFillBytesRejectsNegativeCountBeforeMutation(t *testing.T) {
	for _, count := range []int64{-1, math.MinInt64} {
		size := expression.New()
		size.SetValue(count)
		filler := []byte{0xA5, 0x5A}
		dat := &data{
			size:   size,
			values: []any{filler},
		}
		assert.ErrorContains(t, generateDataFillBytes(dat), "negative")
		assert.Equal(t, []any{filler}, dat.values)
	}
}

func TestGenerateDataFillBytesPreservesZeroAndPositiveCounts(t *testing.T) {
	for _, test := range []struct {
		count int64
		want  []byte
	}{
		{
			count: 0,
			want:  []byte{},
		},
		{
			count: 5,
			want:  []byte{0xA5, 0x5A, 0xA5, 0x5A, 0xA5},
		},
	} {
		size := expression.New()
		size.SetValue(test.count)
		dat := &data{
			size:   size,
			values: []any{[]byte{0xA5, 0x5A}},
		}
		assert.NoError(t, generateDataFillBytes(dat))
		assert.Equal(t, []any{test.want}, dat.values)
	}
}

func TestAssemblerFillRetainsCompleteImageBounds(t *testing.T) {
	for _, count := range []int64{0, 1, 0xE00, 0xE01} {
		fill := ast.NewData(ast.DataType, 1)
		fill.Fill = true
		fill.Size = expression.New(token.Token{
			Type:  token.Number,
			Value: strconv.FormatInt(count, 10),
		})
		var output bytes.Buffer
		assembly := New(asmchip8.New(), &output)
		err := assembly.ProcessAST(t.Context(), []ast.Node{fill})
		if count > 0xE00 {
			assert.Error(t, err)
			assert.Empty(t, output.Bytes())
			continue
		}

		assert.NoError(t, err)
		assert.True(t, bytes.Equal(make([]byte, count), output.Bytes()), "count=%d output=%d", count, output.Len())
	}
}
