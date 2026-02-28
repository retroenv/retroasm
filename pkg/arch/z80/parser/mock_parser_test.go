package parser

import "github.com/retroenv/retroasm/pkg/lexer/token"

type mockParser struct {
	position int
	tokens   []token.Token
}

func newMockParser(tokens ...token.Token) *mockParser {
	return &mockParser{
		tokens: tokens,
	}
}

func (m *mockParser) AddressWidth() int {
	return 16
}

func (m *mockParser) AdvanceReadPosition(offset int) {
	m.position += offset
}

func (m *mockParser) NextToken(offset int) token.Token {
	index := m.position + offset
	if index < 0 || index >= len(m.tokens) {
		return token.Token{Type: token.EOF}
	}
	return m.tokens[index]
}
