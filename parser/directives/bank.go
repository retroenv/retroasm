package directives

import (
	"fmt"

	"github.com/retroenv/retroasm/lexer/token"
	"github.com/retroenv/retroasm/number"
	"github.com/retroenv/retroasm/parser/ast"
)

// Bank ...
func Bank(p Parser) (ast.Node, error) {
	value := p.NextToken(2)
	if value.Type != token.Number {
		return nil, fmt.Errorf("unsupported offset counter value type %s", value.Type)
	}

	i, err := number.Parse(value.Value)
	if err != nil {
		return nil, fmt.Errorf("parsing number '%s': %w", value.Value, err)
	}

	p.AdvanceReadPosition(2)
	return ast.NewBank(int(i)), nil
}
