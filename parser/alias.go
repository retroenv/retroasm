package parser

import (
	"errors"
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/lexer/token"
	"github.com/retroenv/retroasm/parser/ast"
	"github.com/retroenv/retroasm/parser/directives"
)

var errUnsupportedIdentifier = errors.New("unsupported identifier")

func (p *Parser) parseAlias(tok, next token.Token) (ast.Node, error) {
	evaluateOnce := false
	symbolReusable := false

	switch {
	case next.Type == token.Assign:
		evaluateOnce = true
		symbolReusable = true
		break

	case next.Type == token.Identifier && strings.ToUpper(next.Value) == "EQU":
		break

	default:
		// check if token string is a directive and was used without .
		directive := strings.ToLower(tok.Value)
		handler, ok := directives.Handlers[directive]
		if !ok {
			return nil, fmt.Errorf("'%s': %w", tok.Value, errUnsupportedIdentifier)
		}
		p.readPosition-- // advance back since dot handler expects dot token
		return handler(p)
	}

	alias, err := p.parseAliasValues(tok)
	if err != nil {
		return nil, fmt.Errorf("parsing alias values: %w", err)
	}
	alias.SymbolReusable = symbolReusable
	alias.Expression.SetEvaluateOnce(evaluateOnce)

	p.readPosition += 2
	return alias, nil
}

func (p *Parser) parseAliasValues(tok token.Token) (ast.Alias, error) {
	alias := ast.NewAlias(tok.Value)

	tokens := 0

	for next := p.NextToken(2); !next.Type.IsTerminator(); {
		alias.Expression.AddTokens(next)
		tokens++

		next = p.NextToken(3)
		if next.Type.IsTerminator() {
			break
		}
		p.readPosition++
	}

	if tokens == 0 {
		// there needs to be at least one valid node
		return ast.Alias{}, errMissingParameter
	}
	return alias, nil
}
