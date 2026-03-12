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

func TestParserX816NoOpDirectives(t *testing.T) {
	noOpDirectives := []string{
		".mem 8",
		".index 8",
		".opt",
		".optimize",
		".list",
		".nolist",
		".sym",
		".symbol",
		".detect",
		".dasm",
		".echo text",
		".hrom",
		".lrom",
		".hirom",
		".smc",
		".par",
		".parenthesis",
		".localsymbolchar _",
		".locchar _",
		".cerror",
		".cwarn",
		".message text",
	}

	cfg := m6502Arch.New()
	for _, directive := range noOpDirectives {
		p := New(cfg.Arch, strings.NewReader(directive+"\n"), config.CompatX816)
		assert.NoError(t, p.Read(t.Context()), "directive: "+directive)
		nodes, err := p.TokensToAstNodes()
		assert.NoError(t, err, "directive: "+directive)
		assert.Len(t, nodes, 0, "directive: "+directive)
	}
}

func TestParserX816CommentBlock(t *testing.T) {
	input := ".comment\nthis is a comment\nspanning multiple lines\n.end\n"

	cfg := m6502Arch.New()
	p := New(cfg.Arch, strings.NewReader(input), config.CompatX816)
	assert.NoError(t, p.Read(t.Context()))
	nodes, err := p.TokensToAstNodes()
	assert.NoError(t, err)
	assert.Len(t, nodes, 0)
}

func TestParserX816CommentBlockWithCode(t *testing.T) {
	input := "nop\n.comment\nskipped\n.end\nnop\n"

	cfg := m6502Arch.New()
	p := New(cfg.Arch, strings.NewReader(input), config.CompatX816)
	assert.NoError(t, p.Read(t.Context()))
	nodes, err := p.TokensToAstNodes()
	assert.NoError(t, err)

	assert.Len(t, nodes, 2)
	assert.Equal(t, ast.NewInstruction("nop", int(m6502.ImpliedAddressing), nil, nil), nodes[0])
	assert.Equal(t, ast.NewInstruction("nop", int(m6502.ImpliedAddressing), nil, nil), nodes[1])
}

func TestParserX816SourceInclude(t *testing.T) {
	input := ".src test.asm\n"

	cfg := m6502Arch.New()
	p := New(cfg.Arch, strings.NewReader(input), config.CompatX816)
	assert.NoError(t, p.Read(t.Context()))
	nodes, err := p.TokensToAstNodes()
	assert.NoError(t, err)

	assert.Len(t, nodes, 1)
	assert.Equal(t, ast.NewInclude("test.asm", false, 0, 0), nodes[0])
}

func TestParserX816DotEqu(t *testing.T) {
	input := "MAX .equ 255\n"

	cfg := m6502Arch.New()
	p := New(cfg.Arch, strings.NewReader(input), config.CompatX816)
	assert.NoError(t, p.Read(t.Context()))
	nodes, err := p.TokensToAstNodes()
	assert.NoError(t, err)

	assert.Len(t, nodes, 1)
	alias, ok := nodes[0].(ast.Alias)
	assert.True(t, ok)
	assert.Equal(t, "MAX", alias.Name)
}

func TestParserX816ColonOptionalLabel(t *testing.T) {
	input := "start\nnop\n"

	cfg := m6502Arch.New()
	p := New(cfg.Arch, strings.NewReader(input), config.CompatX816)
	assert.NoError(t, p.Read(t.Context()))
	nodes, err := p.TokensToAstNodes()
	assert.NoError(t, err)

	assert.Len(t, nodes, 2)
	assert.Equal(t, ast.NewLabel("start"), nodes[0])
	assert.Equal(t, ast.NewInstruction("nop", int(m6502.ImpliedAddressing), nil, nil), nodes[1])
}

func TestParserX816ColonOptionalLabelBeforeInstruction(t *testing.T) {
	// In x816 mode, label at column 0 followed by instruction on next token
	input := "start\n  nop\n"

	cfg := m6502Arch.New()
	p := New(cfg.Arch, strings.NewReader(input), config.CompatX816)
	assert.NoError(t, p.Read(t.Context()))
	nodes, err := p.TokensToAstNodes()
	assert.NoError(t, err)

	assert.Len(t, nodes, 2)
	assert.Equal(t, ast.NewLabel("start"), nodes[0])
	assert.Equal(t, ast.NewInstruction("nop", int(m6502.ImpliedAddressing), nil, nil), nodes[1])
}

func TestParserX816AnonymousLabel(t *testing.T) {
	input := "+\nnop\n"

	cfg := m6502Arch.New()
	p := New(cfg.Arch, strings.NewReader(input), config.CompatX816)
	assert.NoError(t, p.Read(t.Context()))
	nodes, err := p.TokensToAstNodes()
	assert.NoError(t, err)

	assert.Len(t, nodes, 2)
	// First node is the anonymous label
	label, ok := nodes[0].(ast.Label)
	assert.True(t, ok)
	assert.True(t, strings.HasPrefix(label.Name, "__anon_fwd_"))
}

func TestParserX816EndDirective(t *testing.T) {
	input := ".end\n"

	cfg := m6502Arch.New()
	p := New(cfg.Arch, strings.NewReader(input), config.CompatX816)
	assert.NoError(t, p.Read(t.Context()))
	nodes, err := p.TokensToAstNodes()
	assert.NoError(t, err)
	assert.Len(t, nodes, 0)
}
