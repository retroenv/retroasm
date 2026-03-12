package directives

import (
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

// CommentBlock handles x816-style multi-line comment blocks (.comment ... .end).
// All tokens are skipped until a matching .end directive is found.
//
//nolint:nilnil // directive is intentionally ignored
func CommentBlock(p arch.Parser) (ast.Node, error) {
	// Skip past "comment" identifier
	p.AdvanceReadPosition(1)

	for {
		p.AdvanceReadPosition(1)
		tok := p.NextToken(0)

		if tok.Type == token.EOF {
			return nil, nil
		}

		if tok.Type == token.Dot {
			next := p.NextToken(1)
			if next.Type == token.Identifier && strings.ToLower(next.Value) == "end" {
				p.AdvanceReadPosition(1)
				return nil, nil
			}
		}
	}
}
