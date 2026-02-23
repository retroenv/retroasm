package assembler

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/assert"
)

func TestModifierOffset(t *testing.T) { //nolint:funlen
	tests := []struct {
		name       string
		modifiers  []ast.Modifier
		wantOffset int64
		wantErr    bool
	}{
		{
			name:       "empty modifiers",
			modifiers:  nil,
			wantOffset: 0,
		},
		{
			name: "single addition",
			modifiers: []ast.Modifier{
				{Operator: ast.NewOperator("+"), Value: "5"},
			},
			wantOffset: 5,
		},
		{
			name: "single subtraction",
			modifiers: []ast.Modifier{
				{Operator: ast.NewOperator("-"), Value: "3"},
			},
			wantOffset: -3,
		},
		{
			name: "multiple modifiers cumulative",
			modifiers: []ast.Modifier{
				{Operator: ast.NewOperator("+"), Value: "10"},
				{Operator: ast.NewOperator("-"), Value: "4"},
				{Operator: ast.NewOperator("+"), Value: "1"},
			},
			wantOffset: 7,
		},
		{
			name: "hex value modifier",
			modifiers: []ast.Modifier{
				{Operator: ast.NewOperator("+"), Value: "$10"},
			},
			wantOffset: 16,
		},
		{
			name: "invalid number value",
			modifiers: []ast.Modifier{
				{Operator: ast.NewOperator("+"), Value: "xyz"},
			},
			wantErr: true,
		},
		{
			name: "unsupported operator",
			modifiers: []ast.Modifier{
				{Operator: ast.NewOperator("*"), Value: "2"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := modifierOffset(tt.modifiers)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantOffset, got)
		})
	}
}

func TestNameWithModifiers(t *testing.T) { //nolint:funlen
	tests := []struct {
		name      string
		symName   string
		modifiers []ast.Modifier
		want      string
		wantErr   bool
	}{
		{
			name:      "no modifiers",
			symName:   "label",
			modifiers: nil,
			want:      "label",
		},
		{
			name:    "positive offset",
			symName: "noise",
			modifiers: []ast.Modifier{
				{Operator: ast.NewOperator("+"), Value: "5"},
			},
			want: "noise+5",
		},
		{
			name:    "negative offset",
			symName: "label",
			modifiers: []ast.Modifier{
				{Operator: ast.NewOperator("-"), Value: "3"},
			},
			want: "label-3",
		},
		{
			name:    "zero offset",
			symName: "base",
			modifiers: []ast.Modifier{
				{Operator: ast.NewOperator("+"), Value: "0"},
			},
			want: "base+0",
		},
		{
			name:    "combined modifiers net positive",
			symName: "tileData",
			modifiers: []ast.Modifier{
				{Operator: ast.NewOperator("+"), Value: "8"},
				{Operator: ast.NewOperator("-"), Value: "2"},
			},
			want: "tileData+6",
		},
		{
			name:    "invalid modifier propagates error",
			symName: "label",
			modifiers: []ast.Modifier{
				{Operator: ast.NewOperator("+"), Value: "bad"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nameWithModifiers(tt.symName, tt.modifiers)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseInstruction(t *testing.T) { //nolint:funlen
	tests := []struct {
		name    string
		ins     ast.Instruction
		wantArg any
		wantErr bool
	}{
		{
			name:    "no argument no modifier",
			ins:     ast.NewInstruction("nop", 0, nil, nil),
			wantArg: nil,
		},
		{
			name:    "number argument no modifier",
			ins:     ast.NewInstruction("lda", 0, ast.NewNumber(0xFF), nil),
			wantArg: uint64(0xFF),
		},
		{
			name: "number argument with addition modifier",
			ins: ast.NewInstruction("lda", 0, ast.NewNumber(0x10), []ast.Modifier{
				{Operator: ast.NewOperator("+"), Value: "5"},
			}),
			wantArg: uint64(0x15),
		},
		{
			name: "number argument with subtraction modifier",
			ins: ast.NewInstruction("lda", 0, ast.NewNumber(0x10), []ast.Modifier{
				{Operator: ast.NewOperator("-"), Value: "2"},
			}),
			wantArg: uint64(0x0E),
		},
		{
			name:    "label argument no modifier",
			ins:     ast.NewInstruction("jmp", 0, ast.NewLabel("loop"), nil),
			wantArg: reference{name: "loop"},
		},
		{
			name: "label argument with positive modifier",
			ins: ast.NewInstruction("jmp", 0, ast.NewLabel("tileData"), []ast.Modifier{
				{Operator: ast.NewOperator("+"), Value: "8"},
			}),
			wantArg: reference{name: "tileData+8"},
		},
		{
			name: "label argument with negative modifier",
			ins: ast.NewInstruction("jmp", 0, ast.NewLabel("base"), []ast.Modifier{
				{Operator: ast.NewOperator("-"), Value: "3"},
			}),
			wantArg: reference{name: "base-3"},
		},
		{
			name:    "identifier argument no modifier",
			ins:     ast.NewInstruction("jsr", 0, ast.NewIdentifier("myFunc"), nil),
			wantArg: reference{name: "myFunc"},
		},
		{
			name: "identifier argument with modifier",
			ins: ast.NewInstruction("jsr", 0, ast.NewIdentifier("noise"), []ast.Modifier{
				{Operator: ast.NewOperator("+"), Value: "1"},
			}),
			wantArg: reference{name: "noise+1"},
		},
		{
			name: "invalid modifier on number argument",
			ins: ast.NewInstruction("lda", 0, ast.NewNumber(0x10), []ast.Modifier{
				{Operator: ast.NewOperator("+"), Value: "bad"},
			}),
			wantErr: true,
		},
		{
			name: "invalid modifier on label argument",
			ins: ast.NewInstruction("jmp", 0, ast.NewLabel("label"), []ast.Modifier{
				{Operator: ast.NewOperator("*"), Value: "2"},
			}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parseInstruction(tt.ins)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, nodes, 1)

			ins, ok := nodes[0].(*instruction)
			assert.True(t, ok)
			assert.Equal(t, tt.wantArg, ins.argument)
		})
	}
}
