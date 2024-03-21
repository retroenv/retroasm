package directives

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/retroenv/retroasm/expression"
	"github.com/retroenv/retroasm/lexer/token"
	"github.com/retroenv/retroasm/parser/ast"
)

// Hex ...
func Hex(p Parser) (ast.Node, error) {
	if p.NextToken(2).Type.IsTerminator() {
		return nil, errMissingParameter
	}

	data := ast.NewData(ast.DataType, 1)
	p.AdvanceReadPosition(2)

	var tokens []token.Token

	for {
		tok := p.NextToken(0)
		switch tok.Type {
		case token.Identifier, token.Number:
			s := strings.TrimPrefix(tok.Value, "0x")
			s = strings.TrimPrefix(s, "0X")
			if len(s)%2 != 0 {
				s = "0" + s
			}

			for i := 0; i < len(s); i += 2 {
				w := s[i : i+2]
				i, err := strconv.ParseUint(w, 16, 8)
				if err != nil {
					return nil, fmt.Errorf("parsing hex string '%s': %w", s, err)
				}
				tok.Type = token.Number
				tok.Value = strconv.FormatUint(i, 10)
				tokens = append(tokens, tok)
			}

		default:
			data.Values = expression.New(tokens...)
			return data, nil
		}

		p.AdvanceReadPosition(1)
	}
}
