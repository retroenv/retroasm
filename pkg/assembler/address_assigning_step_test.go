package assembler

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/arch"
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

func TestAddressAssign_RecordInstructionRelocation(t *testing.T) {
	t.Parallel()

	relocations := make([]ast.Relocation, 0)
	aa := addressAssign[any]{instructionRelocations: &relocations}
	ins := &instruction{sourceEntryIndex: 3, hasSourceEntry: true}
	encoding := arch.RelocationEncoding{
		ByteOffset:    1,
		Kind:          ast.RelativeRelocation,
		Width:         ast.WidthByte,
		ByteOrder:     ast.ByteOrderLittle,
		ReferenceType: ast.FullAddress,
		Field:         ast.PackedField{BitWidth: 4, PreserveMask: 0xf0},
	}
	aa.RecordInstructionRelocation(ins, reference{name: "target+2"}, encoding)
	aa.RecordInstructionRelocation(ins, ast.InstructionReference{
		Value:         ast.NewLabel("other"),
		Modifiers:     []ast.Modifier{{Operator: ast.NewOperator("-"), Value: "3"}},
		ReferenceType: ast.FullAddress,
	}, encoding)
	aa.RecordInstructionRelocation(ins, uint64(1), encoding)
	aa.RecordInstructionRelocation(&instruction{}, reference{name: "expanded"}, encoding)

	assert.Equal(t, []ast.Relocation{
		{
			EntryIndex: 3,
			ByteOffset: 1,
			Kind:       ast.RelativeRelocation,
			Expression: ast.NewSymbolExpression("target", 2, ast.FullAddress),
			Width:      ast.WidthByte,
			ByteOrder:  ast.ByteOrderLittle,
			Field:      ast.PackedField{BitWidth: 4, PreserveMask: 0xf0},
		},
		{
			EntryIndex: 3,
			ByteOffset: 1,
			Kind:       ast.RelativeRelocation,
			Expression: ast.NewSymbolExpression("other", -3, ast.FullAddress),
			Width:      ast.WidthByte,
			ByteOrder:  ast.ByteOrderLittle,
			Field:      ast.PackedField{BitWidth: 4, PreserveMask: 0xf0},
		},
	}, relocations)
}
