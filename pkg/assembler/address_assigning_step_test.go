package assembler

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retroasm/pkg/scope"
	"github.com/retroenv/retrogolib/assert"
)

func TestParseReferenceOffset(t *testing.T) {
	tests := []struct {
		input      string
		wantName   string
		wantOffset int64
	}{
		{"symbol", "symbol", 0},
		{"tileData+8", "tileData", 8},
		{"base-3", "base", -3},
		{"my_var+128", "my_var", 128},
		{"nooffset", "nooffset", 0},
		{"a+0", "a", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			name, offset := parseReferenceOffset(tt.input)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantOffset, offset)
		})
	}
}

func TestAssignVariableAddress(t *testing.T) {
	aa := addressAssign[any]{
		programCounter: 0x200,
	}
	v := &variable{
		v: ast.NewVariable("test", 4),
	}

	result := assignVariableAddress(aa, v)
	assert.Equal(t, uint64(0x204), result)
	assert.Equal(t, uint64(0x200), v.address)
}

func TestAddressAssign_ArgumentValueExpression(t *testing.T) {
	aa := addressAssign[any]{
		currentScope:   scope.New(nil),
		programCounter: 0x200,
	}

	t.Run("evaluates arithmetic expression", func(t *testing.T) {
		value, err := aa.ArgumentValue(ast.NewExpression(
			token.Token{Type: token.Number, Value: "1"},
			token.Token{Type: token.Plus},
			token.Token{Type: token.Number, Value: "2"},
		))
		assert.NoError(t, err)
		assert.Equal(t, uint64(3), value)
	})

	t.Run("evaluates program counter expression", func(t *testing.T) {
		value, err := aa.ArgumentValue(ast.NewExpression(
			token.Token{Type: token.Number, Value: "$"},
			token.Token{Type: token.Plus},
			token.Token{Type: token.Number, Value: "1"},
		))
		assert.NoError(t, err)
		assert.Equal(t, uint64(0x201), value)
	})
}
