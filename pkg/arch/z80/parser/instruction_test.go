package parser

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
	"github.com/retroenv/retrogolib/assert"
)

func TestParseIdentifier_MinimumSlice(t *testing.T) { //nolint:funlen
	tests := []struct {
		name           string
		mnemonic       string
		tokens         []token.Token
		variants       []*cpuz80.Instruction
		wantVariant    *cpuz80.Instruction
		wantAddressing cpuz80.AddressingMode
		wantRegister   []cpuz80.RegisterParam
		wantValueType  any
		wantValueNode  ast.Node
	}{
		{
			name:           "nop implied",
			mnemonic:       cpuz80.NopName,
			tokens:         []token.Token{{Type: token.Identifier, Value: "nop"}, {Type: token.EOL}},
			variants:       []*cpuz80.Instruction{cpuz80.Nop},
			wantVariant:    cpuz80.Nop,
			wantAddressing: cpuz80.ImpliedAddressing,
		},
		{
			name:           "ret implied selects unconditional variant",
			mnemonic:       cpuz80.RetName,
			tokens:         []token.Token{{Type: token.Identifier, Value: "ret"}, {Type: token.EOL}},
			variants:       []*cpuz80.Instruction{cpuz80.RetCond, cpuz80.Ret},
			wantVariant:    cpuz80.Ret,
			wantAddressing: cpuz80.ImpliedAddressing,
		},
		{
			name:           "ld register pair",
			mnemonic:       cpuz80.LdName,
			tokens:         []token.Token{{Type: token.Identifier, Value: "ld"}, {Type: token.Identifier, Value: "a"}, {Type: token.Comma}, {Type: token.Identifier, Value: "b"}, {Type: token.EOL}},
			variants:       []*cpuz80.Instruction{cpuz80.LdImm8, cpuz80.LdReg8, cpuz80.LdReg16},
			wantVariant:    cpuz80.LdReg8,
			wantAddressing: cpuz80.RegisterAddressing,
			wantRegister:   []cpuz80.RegisterParam{cpuz80.RegA, cpuz80.RegB},
		},
		{
			name:           "ld a immediate",
			mnemonic:       cpuz80.LdName,
			tokens:         []token.Token{{Type: token.Identifier, Value: "ld"}, {Type: token.Identifier, Value: "a"}, {Type: token.Comma}, {Type: token.Number, Value: "42"}, {Type: token.EOL}},
			variants:       []*cpuz80.Instruction{cpuz80.LdImm8, cpuz80.LdReg8, cpuz80.LdReg16},
			wantVariant:    cpuz80.LdImm8,
			wantAddressing: cpuz80.ImmediateAddressing,
			wantRegister:   []cpuz80.RegisterParam{cpuz80.RegA},
			wantValueType:  ast.Number{},
			wantValueNode:  ast.NewNumber(42),
		},
		{
			name:           "ld hl immediate 16-bit value",
			mnemonic:       cpuz80.LdName,
			tokens:         []token.Token{{Type: token.Identifier, Value: "ld"}, {Type: token.Identifier, Value: "hl"}, {Type: token.Comma}, {Type: token.Number, Value: "$1234"}, {Type: token.EOL}},
			variants:       []*cpuz80.Instruction{cpuz80.LdImm8, cpuz80.LdReg8, cpuz80.LdReg16},
			wantVariant:    cpuz80.LdReg16,
			wantAddressing: cpuz80.ImmediateAddressing,
			wantRegister:   []cpuz80.RegisterParam{cpuz80.RegHL},
			wantValueType:  ast.Number{},
			wantValueNode:  ast.NewNumber(0x1234),
		},
		{
			name:           "jr relative",
			mnemonic:       cpuz80.JrName,
			tokens:         []token.Token{{Type: token.Identifier, Value: "jr"}, {Type: token.Identifier, Value: "loop"}, {Type: token.EOL}},
			variants:       []*cpuz80.Instruction{cpuz80.JrCond, cpuz80.JrRel},
			wantVariant:    cpuz80.JrRel,
			wantAddressing: cpuz80.RelativeAddressing,
			wantValueType:  ast.Label{},
			wantValueNode:  ast.NewLabel("loop"),
		},
		{
			name:           "jr conditional relative",
			mnemonic:       cpuz80.JrName,
			tokens:         []token.Token{{Type: token.Identifier, Value: "jr"}, {Type: token.Identifier, Value: "nz"}, {Type: token.Comma}, {Type: token.Identifier, Value: "loop"}, {Type: token.EOL}},
			variants:       []*cpuz80.Instruction{cpuz80.JrRel, cpuz80.JrCond},
			wantVariant:    cpuz80.JrCond,
			wantAddressing: cpuz80.RelativeAddressing,
			wantRegister:   []cpuz80.RegisterParam{cpuz80.RegCondNZ},
			wantValueType:  ast.Label{},
			wantValueNode:  ast.NewLabel("loop"),
		},
		{
			name:           "jp absolute",
			mnemonic:       cpuz80.JpName,
			tokens:         []token.Token{{Type: token.Identifier, Value: "jp"}, {Type: token.Identifier, Value: "target"}, {Type: token.EOL}},
			variants:       []*cpuz80.Instruction{cpuz80.JpCond, cpuz80.JpAbs},
			wantVariant:    cpuz80.JpAbs,
			wantAddressing: cpuz80.ExtendedAddressing,
			wantValueType:  ast.Label{},
			wantValueNode:  ast.NewLabel("target"),
		},
		{
			name:           "jp conditional with c uses condition code",
			mnemonic:       cpuz80.JpName,
			tokens:         []token.Token{{Type: token.Identifier, Value: "jp"}, {Type: token.Identifier, Value: "c"}, {Type: token.Comma}, {Type: token.Identifier, Value: "target"}, {Type: token.EOL}},
			variants:       []*cpuz80.Instruction{cpuz80.JpAbs, cpuz80.JpCond},
			wantVariant:    cpuz80.JpCond,
			wantAddressing: cpuz80.ExtendedAddressing,
			wantRegister:   []cpuz80.RegisterParam{cpuz80.RegCondC},
			wantValueType:  ast.Label{},
			wantValueNode:  ast.NewLabel("target"),
		},
		{
			name:           "call absolute",
			mnemonic:       cpuz80.CallName,
			tokens:         []token.Token{{Type: token.Identifier, Value: "call"}, {Type: token.Identifier, Value: "target"}, {Type: token.EOL}},
			variants:       []*cpuz80.Instruction{cpuz80.CallCond, cpuz80.Call},
			wantVariant:    cpuz80.Call,
			wantAddressing: cpuz80.ExtendedAddressing,
			wantValueType:  ast.Label{},
			wantValueNode:  ast.NewLabel("target"),
		},
		{
			name:           "bit value first register",
			mnemonic:       cpuz80.BitName,
			tokens:         []token.Token{{Type: token.Identifier, Value: "bit"}, {Type: token.Number, Value: "3"}, {Type: token.Comma}, {Type: token.Identifier, Value: "a"}, {Type: token.EOL}},
			variants:       []*cpuz80.Instruction{cpuz80.CBBit},
			wantVariant:    cpuz80.CBBit,
			wantAddressing: cpuz80.RegisterAddressing,
			wantRegister:   []cpuz80.RegisterParam{cpuz80.RegA},
			wantValueType:  ast.Number{},
			wantValueNode:  ast.NewNumber(3),
		},
		{
			name:           "im numeric register opcode variant",
			mnemonic:       cpuz80.ImName,
			tokens:         []token.Token{{Type: token.Identifier, Value: "im"}, {Type: token.Number, Value: "1"}, {Type: token.EOL}},
			variants:       []*cpuz80.Instruction{cpuz80.EdIm0, cpuz80.EdIm1, cpuz80.EdIm2},
			wantVariant:    cpuz80.EdIm1,
			wantAddressing: cpuz80.ImmediateAddressing,
			wantRegister:   []cpuz80.RegisterParam{cpuz80.RegIM1},
		},
		{
			name:           "rst numeric register opcode variant",
			mnemonic:       cpuz80.RstName,
			tokens:         []token.Token{{Type: token.Identifier, Value: "rst"}, {Type: token.Number, Value: "$38"}, {Type: token.EOL}},
			variants:       []*cpuz80.Instruction{cpuz80.Rst},
			wantVariant:    cpuz80.Rst,
			wantAddressing: cpuz80.ImpliedAddressing,
			wantRegister:   []cpuz80.RegisterParam{cpuz80.RegRst38},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := newMockParser(tt.tokens...)
			node, err := ParseIdentifier(parser, tt.mnemonic, tt.variants)
			assert.NoError(t, err)

			ins, ok := node.(ast.Instruction)
			assert.True(t, ok)
			assert.Equal(t, tt.mnemonic, ins.Name)
			assert.Equal(t, int(tt.wantAddressing), ins.Addressing)

			typedArg, ok := ins.Argument.(ast.InstructionArgument)
			assert.True(t, ok)

			resolved, ok := typedArg.Value.(ResolvedInstruction)
			assert.True(t, ok)
			assert.Equal(t, tt.wantVariant, resolved.Instruction)
			assert.Equal(t, tt.wantAddressing, resolved.Addressing)
			assert.Equal(t, tt.wantRegister, resolved.RegisterParams)

			if tt.wantValueType == nil {
				assert.Empty(t, resolved.OperandValues)
				return
			}

			assert.Len(t, resolved.OperandValues, 1)
			assert.Equal(t, tt.wantValueNode, resolved.OperandValues[0])
		})
	}
}

func TestParseIdentifier_Errors(t *testing.T) {
	tests := []struct {
		name     string
		mnemonic string
		tokens   []token.Token
		variants []*cpuz80.Instruction
	}{
		{
			name:     "missing second operand after comma",
			mnemonic: cpuz80.LdName,
			tokens:   []token.Token{{Type: token.Identifier, Value: "ld"}, {Type: token.Identifier, Value: "a"}, {Type: token.Comma}, {Type: token.EOL}},
			variants: []*cpuz80.Instruction{cpuz80.LdImm8, cpuz80.LdReg8},
		},
		{
			name:     "unsupported operand pattern",
			mnemonic: cpuz80.JpName,
			tokens:   []token.Token{{Type: token.Identifier, Value: "jp"}, {Type: token.Number, Value: "1"}, {Type: token.Comma}, {Type: token.Number, Value: "2"}, {Type: token.EOL}},
			variants: []*cpuz80.Instruction{cpuz80.JpAbs, cpuz80.JpCond},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := newMockParser(tt.tokens...)
			_, err := ParseIdentifier(parser, tt.mnemonic, tt.variants)
			assert.Error(t, err)
		})
	}
}
