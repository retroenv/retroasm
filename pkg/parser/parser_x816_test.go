package parser

import (
	"strings"
	"testing"

	m6502Arch "github.com/retroenv/retroasm/pkg/arch/m6502"
	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/m6502"
	"github.com/retroenv/retrogolib/assert"
)

var x816NoOpDirectives = []string{
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
	".end",
}

var x816DataWidths = []struct {
	directive string
	want      int
}{
	{directive: "dcl", want: 3},
	{directive: "dl", want: 3},
	{directive: "dcd", want: 4},
	{directive: "dd", want: 4},
	{directive: "dsl", want: 3},
	{directive: "dsd", want: 4},
}

func TestParserX816NoOpDirectives(t *testing.T) {
	for _, directive := range x816NoOpDirectives {
		t.Run(directive, func(t *testing.T) {
			nodes := parseX816(t, directive+"\n")
			assert.Empty(t, nodes)
		})
	}
}

func TestParserX816CommentBlock(t *testing.T) {
	nodes := parseX816(t, "nop\n.comment\nskipped\n.end\nnop\n")

	assert.Len(t, nodes, 2)
	assert.Equal(t, m6502Instruction("nop", int(m6502.ImpliedAddressing), nil), nodes[0])
	assert.Equal(t, m6502Instruction("nop", int(m6502.ImpliedAddressing), nil), nodes[1])
}

func TestParserX816CommentBlock_Unterminated(t *testing.T) {
	nodes := parseX816(t, ".comment\nskipped\n")
	assert.Empty(t, nodes)
}

func TestParserX816SourceInclude(t *testing.T) {
	nodes := parseX816(t, ".src test.asm\n")

	assert.Len(t, nodes, 1)
	assert.Equal(t, ast.NewInclude("test.asm", false, 0, 0), nodes[0])
}

func TestParserX816DotEqu(t *testing.T) {
	nodes := parseX816(t, "MAX .equ 255\n")

	assert.Len(t, nodes, 1)
	alias, ok := nodes[0].(ast.Alias)
	assert.True(t, ok)
	assert.Equal(t, "MAX", alias.Name)
}

func TestParserX816ColonOptionalLabel(t *testing.T) {
	nodes := parseX816(t, "start\n  nop\n")

	assert.Len(t, nodes, 2)
	assert.Equal(t, ast.NewLabel("start"), nodes[0])
	assert.Equal(t, m6502Instruction("nop", int(m6502.ImpliedAddressing), nil), nodes[1])
}

func TestParserX816AnonymousLabels(t *testing.T) {
	nodes := parseX816(t, "+\n++\n-\n--\n")

	assert.Len(t, nodes, 4)
	assert.Equal(t, ast.NewLabel("__anon_fwd_1_1"), nodes[0])
	assert.Equal(t, ast.NewLabel("__anon_fwd_2_2"), nodes[1])
	assert.Equal(t, ast.NewLabel("__anon_bwd_1_1"), nodes[2])
	assert.Equal(t, ast.NewLabel("__anon_bwd_2_2"), nodes[3])
}

func TestParserX816AsteriskProgramCounter(t *testing.T) {
	nodes := parseX816(t, "* = $8000\n")

	assert.Len(t, nodes, 1)
	base, ok := nodes[0].(ast.Base)
	assert.True(t, ok)
	assert.Equal(t, "$8000", base.Address.Tokens()[0].Value)
}

func TestParserX816ImmediateSymbolStartingWithH(t *testing.T) {
	nodes := parseX816(t, "HammerBro = $05\n  cmp #HammerBro\n")

	assert.Len(t, nodes, 2)
	assert.Equal(t,
		m6502Instruction("cmp", int(m6502.ImmediateAddressing), ast.NewIdentifier("HammerBro")),
		nodes[1],
	)
}

func TestParserX816ImmediateAddressBytes(t *testing.T) {
	nodes := parseX816(t, "TitleScreenDataOffset = $1ec0\n  lda #>TitleScreenDataOffset\n  lda #<TitleScreenDataOffset\n  lda #^TitleScreenDataOffset\n")

	assert.Len(t, nodes, 4)
	assertX816ImmediateAddressByte(t, nodes[1], token.Gt)
	assertX816ImmediateAddressByte(t, nodes[2], token.Lt)
	assertX816ImmediateAddressByte(t, nodes[3], token.Caret)
}

func TestParserX816ImmediateSymbolExpression(t *testing.T) {
	nodes := parseX816(t, "A_Button = %10000000\nStart_Button = %00010000\n  cmp #A_Button+Start_Button\n")

	assert.Len(t, nodes, 3)
	assertX816ImmediateExpression(t, nodes[2], "A_Button", token.Plus, token.Identifier, "Start_Button")
}

func TestParserX816DataWidths(t *testing.T) {
	for _, test := range x816DataWidths {
		t.Run(test.directive, func(t *testing.T) {
			nodes := parseX816(t, "."+test.directive+" 1\n")
			assert.Len(t, nodes, 1)

			data, ok := nodes[0].(ast.Data)
			assert.True(t, ok)
			assert.Equal(t, test.want, data.Width)
		})
	}
}

func assertX816ImmediateExpression(
	t *testing.T,
	node ast.Node,
	left string,
	operator, rightType token.Type,
	right string,
) {

	t.Helper()

	instruction, ok := node.(ast.Instruction)
	assert.True(t, ok)
	assert.Equal(t, int(m6502.ImmediateAddressing), instruction.Addressing)

	expression, ok := instruction.Argument.(ast.Expression)
	assert.True(t, ok)
	tokens := expression.Value.Tokens()
	assert.Len(t, tokens, 3)
	assert.Equal(t, token.Identifier, tokens[0].Type)
	assert.Equal(t, left, tokens[0].Value)
	assert.Equal(t, operator, tokens[1].Type)
	assert.Equal(t, rightType, tokens[2].Type)
	assert.Equal(t, right, tokens[2].Value)
}

func assertX816ImmediateAddressByte(t *testing.T, node ast.Node, prefix token.Type) {
	t.Helper()

	instruction, ok := node.(ast.Instruction)
	assert.True(t, ok)
	assert.Equal(t, int(m6502.ImmediateAddressing), instruction.Addressing)

	expression, ok := instruction.Argument.(ast.Expression)
	assert.True(t, ok)
	tokens := expression.Value.Tokens()
	assert.Len(t, tokens, 2)
	assert.Equal(t, prefix, tokens[0].Type)
	assert.Equal(t, token.Identifier, tokens[1].Type)
	assert.Equal(t, "TitleScreenDataOffset", tokens[1].Value)
}

func parseX816(t *testing.T, input string) []ast.Node {
	t.Helper()

	cfg := m6502Arch.New()
	p := New(cfg.Arch, strings.NewReader(input), config.CompatX816)
	assert.NoError(t, p.Read(t.Context()))
	nodes, err := p.TokensToAstNodes()
	assert.NoError(t, err)
	return nodes
}
