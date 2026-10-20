package parser_test

import (
	"strings"
	"testing"

	asmz80 "github.com/retroenv/retroasm/pkg/arch/z80"
	z80parser "github.com/retroenv/retroasm/pkg/arch/z80/parser"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	asmparser "github.com/retroenv/retroasm/pkg/parser"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/assert"
)

func TestZ80DotSymbolOperands(t *testing.T) {
	for _, test := range []struct {
		source string
		value  string
	}{
		{source: "ld hl,.Lretrogo_rodata_0123", value: ".Lretrogo_rodata_0123"},
		{source: "ld hl,.Ltable+2", value: ".Ltable+0x2"},
		{source: "ld a,(.Ltable)", value: ".Ltable"},
		{source: "ld a,(.Ltable+2)", value: ".Ltable+0x2"},
		{source: "ld (.Ltable-1),a", value: ".Ltable-0x1"},
		{source: "jp .Lexit", value: ".Lexit"},
		{source: "jr nz,.Lrepeat", value: ".Lrepeat"},
	} {
		t.Run(test.source, func(t *testing.T) {
			instruction := parseInstruction(t, asmz80.New().Arch, test.source)
			argument, ok := instruction.Argument.(ast.InstructionArgument)
			assert.True(t, ok)
			resolved, ok := argument.Value.(z80parser.ResolvedInstruction)
			assert.True(t, ok)
			assert.Len(t, resolved.OperandValues, 1)
			value, err := ast.FormatValue(resolved.OperandValues[0], ast.ValueFormatOptions{})
			assert.NoError(t, err)
			assert.Equal(t, test.value, value)
		})
	}
}

func TestZ80DotSymbolRejectsMissingIdentifier(t *testing.T) {
	for _, source := range []string{"ld hl,.", "ld a,(.)", "jp .1"} {
		t.Run(source, func(t *testing.T) {
			parser := asmparser.New(asmz80.New().Arch, strings.NewReader(source), config.CompatDefault)
			assert.NoError(t, parser.Read(t.Context()))
			_, err := parser.TokensToAstNodes()
			assert.Error(t, err)
		})
	}
}
