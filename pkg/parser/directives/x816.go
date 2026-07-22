package directives

import (
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

// CommentBlock consumes an x816 comment block through its .end marker.
//
//nolint:nilnil // comment blocks intentionally produce no AST node
func CommentBlock(p arch.Parser) (ast.Node, error) {
	p.AdvanceReadPosition(1)

	for {
		p.AdvanceReadPosition(1)
		tok := p.NextToken(0)
		if tok.Type == token.EOF {
			return nil, nil
		}
		if tok.Type != token.Dot {
			continue
		}

		next := p.NextToken(1)
		if next.Type == token.Identifier && strings.EqualFold(next.Value, "end") {
			p.AdvanceReadPosition(1)
			return nil, nil
		}
	}
}

func x816Handlers() map[string]Handler {
	// This overlay both remaps dialect-specific spellings and accepts directives
	// that affect x816 listing/output modes but emit no bytes in this assembler.
	return map[string]Handler{
		"cerror":          NoOp,
		"comment":         CommentBlock,
		"cwarn":           NoOp,
		"dasm":            NoOp,
		"dcd":             Data,
		"dcl":             Data,
		"dd":              Data,
		"detect":          NoOp,
		"dl":              Data,
		"dsd":             DataStorage,
		"dsl":             DataStorage,
		"echo":            NoOp,
		"end":             NoOp,
		"hirom":           NoOp,
		"hrom":            NoOp,
		"index":           NoOp,
		"list":            NoOp,
		"localsymbolchar": NoOp,
		"locchar":         NoOp,
		"lrom":            NoOp,
		"mem":             NoOp,
		"message":         NoOp,
		"nolist":          NoOp,
		"opt":             NoOp,
		"optimize":        NoOp,
		"par":             NoOp,
		"parenthesis":     NoOp,
		"smc":             NoOp,
		"src":             Include,
		"sym":             NoOp,
		"symbol":          NoOp,
	}
}
