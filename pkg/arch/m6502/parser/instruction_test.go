package parser

import (
	"testing"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retrogolib/assert"
)

var resolveArg1TokenTests = []struct {
	name             string
	tokens           []token.Token
	scopePrefix      string
	dotName          string
	unnamedName      string
	wantType         token.Type
	wantValue        string
	wantPosition     int
	wantForward      bool
	wantUnnamedLevel int
}{
	{
		name:         "scoped identifier",
		tokens:       []token.Token{{Type: token.Identifier, Value: "@loop"}},
		scopePrefix:  "main.",
		wantType:     token.Identifier,
		wantValue:    "main.@loop",
		wantPosition: 0,
	},
	{
		name: "multi-level unnamed reference",
		tokens: []token.Token{
			{Type: token.Colon},
			{Type: token.Plus},
			{Type: token.Plus},
			{Type: token.EOL},
		},
		unnamedName:      "__unnamed_2",
		wantType:         token.Identifier,
		wantValue:        "__unnamed_2",
		wantPosition:     2,
		wantForward:      true,
		wantUnnamedLevel: 2,
	},
	{
		name: "dot-local reference",
		tokens: []token.Token{
			{Type: token.Dot},
			{Type: token.Identifier, Value: "loop"},
			{Type: token.EOL},
		},
		dotName:      "main.loop",
		wantType:     token.Identifier,
		wantValue:    "main.loop",
		wantPosition: 1,
	},
	{
		name: "unsupported unnamed reference",
		tokens: []token.Token{
			{Type: token.Colon},
			{Type: token.Minus},
			{Type: token.EOL},
		},
		wantType:         token.Colon,
		wantPosition:     0,
		wantUnnamedLevel: 1,
	},
}

type resolverParser struct {
	tokens         []token.Token
	position       int
	scopePrefix    string
	dotName        string
	unnamedName    string
	unnamedForward bool
	unnamedLevel   int
}

func (p *resolverParser) AddressWidth() int {
	return 16
}

func (p *resolverParser) AdvanceReadPosition(offset int) {
	p.position += offset
}

func (p *resolverParser) NextToken(offset int) token.Token {
	position := p.position + offset
	if position >= len(p.tokens) {
		return token.Token{Type: token.EOF}
	}
	return p.tokens[position]
}

func (p *resolverParser) ResolveDotLocalLabel(_ string) string {
	return p.dotName
}

func (p *resolverParser) ResolveUnnamedLabel(forward bool, level int) string {
	p.unnamedForward = forward
	p.unnamedLevel = level
	return p.unnamedName
}

func (p *resolverParser) ScopeLocalLabel(name string) string {
	return p.scopePrefix + name
}

func TestResolveArg1Token(t *testing.T) {
	for _, test := range resolveArg1TokenTests {
		t.Run(test.name, func(t *testing.T) {
			parser := &resolverParser{
				tokens:      test.tokens,
				scopePrefix: test.scopePrefix,
				dotName:     test.dotName,
				unnamedName: test.unnamedName,
			}

			got := resolveArg1Token(parser)

			assert.Equal(t, test.wantType, got.Type)
			assert.Equal(t, test.wantValue, got.Value)
			assert.Equal(t, test.wantPosition, parser.position)
			assert.Equal(t, test.wantForward, parser.unnamedForward)
			assert.Equal(t, test.wantUnnamedLevel, parser.unnamedLevel)
		})
	}
}
