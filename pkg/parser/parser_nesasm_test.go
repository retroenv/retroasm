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

func TestParserNesasmDotLocalLabelDefinition(t *testing.T) {
	input := "main:\n.loop:\n  dex\n  bne .loop\n"

	cfg := m6502Arch.New()
	p := New(cfg.Arch, strings.NewReader(input), config.CompatNesasm)
	assert.NoError(t, p.Read(t.Context()))
	nodes, err := p.TokensToAstNodes()
	assert.NoError(t, err)

	// Expected: main, main.loop, dex, bne main.loop
	assert.Len(t, nodes, 4)
	assert.Equal(t, ast.NewLabel("main"), nodes[0])
	assert.Equal(t, ast.NewLabel("main.loop"), nodes[1])
	assert.Equal(t, ast.NewInstruction("dex", int(m6502.ImpliedAddressing), nil, nil), nodes[2])
	assert.Equal(t, ast.NewInstruction("bne", int(m6502.RelativeAddressing),
		ast.NewLabel("main.loop"), nil), nodes[3])
}

func TestParserNesasmDotLocalLabelScoping(t *testing.T) {
	input := "sub1:\n.tmp:\nsub2:\n.tmp:\n"

	cfg := m6502Arch.New()
	p := New(cfg.Arch, strings.NewReader(input), config.CompatNesasm)
	assert.NoError(t, p.Read(t.Context()))
	nodes, err := p.TokensToAstNodes()
	assert.NoError(t, err)

	assert.Len(t, nodes, 4)
	assert.Equal(t, ast.NewLabel("sub1"), nodes[0])
	assert.Equal(t, ast.NewLabel("sub1.tmp"), nodes[1])
	assert.Equal(t, ast.NewLabel("sub2"), nodes[2])
	assert.Equal(t, ast.NewLabel("sub2.tmp"), nodes[3])
}

func TestParserNesasmMacroDefinition(t *testing.T) {
	input := "add_val .macro\n  clc\n  adc \\1\n.endm\n"

	cfg := m6502Arch.New()
	p := New(cfg.Arch, strings.NewReader(input), config.CompatNesasm)
	assert.NoError(t, p.Read(t.Context()))
	nodes, err := p.TokensToAstNodes()
	assert.NoError(t, err)

	assert.Len(t, nodes, 1)
	macro, ok := nodes[0].(ast.Macro)
	assert.True(t, ok)
	assert.Equal(t, "add_val", macro.Name)
	assert.Len(t, macro.Arguments, 0) // NESASM macros have no named arguments
	assert.True(t, len(macro.Token) > 0)
}

func TestParserNesasmNoOpDirectives(t *testing.T) {
	noOpDirectives := []string{
		".list",
		".nolist",
		".mlist",
		".nomlist",
		".opt l+",
		".zp",
		".bss",
		".code",
		".data",
	}

	cfg := m6502Arch.New()
	for _, directive := range noOpDirectives {
		p := New(cfg.Arch, strings.NewReader(directive+"\n"), config.CompatNesasm)
		assert.NoError(t, p.Read(t.Context()), "directive: "+directive)
		nodes, err := p.TokensToAstNodes()
		assert.NoError(t, err, "directive: "+directive)
		assert.Len(t, nodes, 0, "directive: "+directive)
	}
}

func TestParserNesasmFail(t *testing.T) {
	input := ".fail \"assembly error\"\n"

	cfg := m6502Arch.New()
	p := New(cfg.Arch, strings.NewReader(input), config.CompatNesasm)
	assert.NoError(t, p.Read(t.Context()))
	nodes, err := p.TokensToAstNodes()
	assert.NoError(t, err)

	assert.Len(t, nodes, 1)
	errNode, ok := nodes[0].(ast.Error)
	assert.True(t, ok)
	assert.Equal(t, "assembly error", errNode.Message)
}

func TestParserNesasmDotLocalLabelNotInDefaultMode(t *testing.T) {
	// In default mode, .unknown should be an error, not a local label
	input := ".unknown\n"

	cfg := m6502Arch.New()
	p := New(cfg.Arch, strings.NewReader(input), config.CompatDefault)
	assert.NoError(t, p.Read(t.Context()))
	_, err := p.TokensToAstNodes()
	assert.Error(t, err)
}
