package directives

import (
	"fmt"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

// Macro parses a .macro directive for defining a reusable code block.
func Macro(p arch.Parser) (ast.Node, error) {
	value := p.NextToken(2)
	if value.Type != token.Identifier {
		return nil, fmt.Errorf("unsupported macro name type %s", value.Type)
	}

	p.AdvanceReadPosition(3)
	m := ast.NewMacro(value.Value)

	// read arguments
	for end := false; !end; {
		tok := p.NextToken(0)
		switch tok.Type {
		case token.Identifier:
			m.Arguments = append(m.Arguments, tok.Value)
		case token.Comma:
		default:
			end = true
			continue
		}

		p.AdvanceReadPosition(1)
	}

	// read all macro tokens
	for end := false; !end; {
		tok := p.NextToken(0)
		p.AdvanceReadPosition(1)

		switch tok.Type {
		case token.EOF:
			end = true
			continue

		case token.Identifier:
			if tok.Value == "ENDM" {
				end = true
				continue
			}
		}

		m.Token = append(m.Token, tok)
	}

	return m, nil
}
