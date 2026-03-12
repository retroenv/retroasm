package parser

import (
	"strings"
	"testing"

	m6502Arch "github.com/retroenv/retroasm/pkg/arch/m6502"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/m6502"
	"github.com/retroenv/retrogolib/assert"
)

func TestParserCa65UnnamedLabelDefinition(t *testing.T) {
	input := ":\n:\n"

	cfg := m6502Arch.New()
	parser := New(cfg.Arch, strings.NewReader(input), config.CompatCa65)
	assert.NoError(t, parser.Read(t.Context()))
	nodes, err := parser.TokensToAstNodes()
	assert.NoError(t, err)

	assert.Len(t, nodes, 2)
	assert.Equal(t, ast.NewLabel("__unnamed_1"), nodes[0])
	assert.Equal(t, ast.NewLabel("__unnamed_2"), nodes[1])
}

func TestParserCa65UnnamedLabelReference(t *testing.T) {
	input := ":\n  bne :-\n:\n  bne :+\n:\n"

	cfg := m6502Arch.New()
	parser := New(cfg.Arch, strings.NewReader(input), config.CompatCa65)
	assert.NoError(t, parser.Read(t.Context()))
	nodes, err := parser.TokensToAstNodes()
	assert.NoError(t, err)

	// Expected: unnamed_1, bne __unnamed_1, unnamed_2, bne __unnamed_3, unnamed_3
	assert.Len(t, nodes, 5)
	assert.Equal(t, ast.NewLabel("__unnamed_1"), nodes[0])
	assert.Equal(t, ast.NewInstruction("bne", int(m6502.RelativeAddressing),
		ast.NewLabel("__unnamed_1"), nil), nodes[1])
	assert.Equal(t, ast.NewLabel("__unnamed_2"), nodes[2])
	assert.Equal(t, ast.NewInstruction("bne", int(m6502.RelativeAddressing),
		ast.NewLabel("__unnamed_3"), nil), nodes[3])
	assert.Equal(t, ast.NewLabel("__unnamed_3"), nodes[4])
}

func TestParserCa65LocalLabelScoping(t *testing.T) {
	input := localLabelScopingInput

	cfg := m6502Arch.New()
	parser := New(cfg.Arch, strings.NewReader(input), config.CompatCa65)
	assert.NoError(t, parser.Read(t.Context()))
	nodes, err := parser.TokensToAstNodes()
	assert.NoError(t, err)

	// ca65 also supports @local scoping
	assert.Len(t, nodes, 4)
	assert.Equal(t, ast.NewLabel("label1"), nodes[0])
	assert.Equal(t, ast.NewLabel("label1.@tmp"), nodes[1])
	assert.Equal(t, ast.NewLabel("label2"), nodes[2])
	assert.Equal(t, ast.NewLabel("label2.@tmp"), nodes[3])
}

func TestParserCa65Scope(t *testing.T) {
	input := ".scope MyScope\n.endscope\n"

	cfg := m6502Arch.New()
	parser := New(cfg.Arch, strings.NewReader(input), config.CompatCa65)
	assert.NoError(t, parser.Read(t.Context()))
	nodes, err := parser.TokensToAstNodes()
	assert.NoError(t, err)

	assert.Len(t, nodes, 2)
	assert.Equal(t, ast.NewScope("MyScope"), nodes[0])
	assert.Equal(t, ast.NewScopeEnd(), nodes[1])
}

func TestParserCa65AnonymousScope(t *testing.T) {
	input := ".scope\n.endscope\n"

	cfg := m6502Arch.New()
	parser := New(cfg.Arch, strings.NewReader(input), config.CompatCa65)
	assert.NoError(t, parser.Read(t.Context()))
	nodes, err := parser.TokensToAstNodes()
	assert.NoError(t, err)

	assert.Len(t, nodes, 2)
	assert.Equal(t, ast.NewScope(""), nodes[0])
	assert.Equal(t, ast.NewScopeEnd(), nodes[1])
}

func TestParserCa65Asciiz(t *testing.T) {
	input := ".asciiz \"hello\"\n"

	cfg := m6502Arch.New()
	parser := New(cfg.Arch, strings.NewReader(input), config.CompatCa65)
	assert.NoError(t, parser.Read(t.Context()))
	nodes, err := parser.TokensToAstNodes()
	assert.NoError(t, err)

	assert.Len(t, nodes, 1)
	data, ok := nodes[0].(ast.Data)
	assert.True(t, ok)
	assert.Equal(t, ast.DataType, data.Type)
	assert.Equal(t, 1, data.Width)
	// Values should include the string tokens plus a trailing 0
	tokens := data.Values.Tokens()
	assert.True(t, len(tokens) > 0)
	// Last token should be the null terminator
	lastToken := tokens[len(tokens)-1]
	assert.Equal(t, "0", lastToken.Value)
}

func TestParserCa65Warning(t *testing.T) {
	input := ".warning \"test message\"\n"

	cfg := m6502Arch.New()
	parser := New(cfg.Arch, strings.NewReader(input), config.CompatCa65)
	assert.NoError(t, parser.Read(t.Context()))
	nodes, err := parser.TokensToAstNodes()
	assert.NoError(t, err)

	assert.Len(t, nodes, 1)
	errNode, ok := nodes[0].(ast.Error)
	assert.True(t, ok)
	assert.Equal(t, "test message", errNode.Message)
}

func TestParserCa65EndMacro(t *testing.T) {
	input := ".macro test_macro arg1\n  lda arg1\n.endmacro\n"

	cfg := m6502Arch.New()
	parser := New(cfg.Arch, strings.NewReader(input), config.CompatCa65)
	assert.NoError(t, parser.Read(t.Context()))
	nodes, err := parser.TokensToAstNodes()
	assert.NoError(t, err)

	assert.Len(t, nodes, 1)
	macro, ok := nodes[0].(ast.Macro)
	assert.True(t, ok)
	assert.Equal(t, "test_macro", macro.Name)
	assert.Len(t, macro.Arguments, 1)
	assert.Equal(t, "arg1", macro.Arguments[0])
}

func TestParserCa65NoOpDirectives(t *testing.T) {
	noOpDirectives := []string{
		".export foo",
		".exportzp foo",
		".import foo",
		".importzp foo",
		".global foo",
		".globalzp foo",
		".feature at_in_identifiers",
		".charmap $41, $61",
		".autoimport +",
		".local foo",
		".debuginfo on",
		".list on",
		".listbytes 8",
		".linecont +",
		".condes foo, 0",
		".define FOO 42",
		".undefine FOO",
		".assert 1, error, \"msg\"",
	}

	cfg := m6502Arch.New()
	for _, directive := range noOpDirectives {
		parser := New(cfg.Arch, strings.NewReader(directive+"\n"), config.CompatCa65)
		assert.NoError(t, parser.Read(t.Context()), "directive: "+directive)
		nodes, err := parser.TokensToAstNodes()
		assert.NoError(t, err, "directive: "+directive)
		assert.Len(t, nodes, 0, "directive: "+directive)
	}
}
