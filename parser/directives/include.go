package directives

import (
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/arch"
	"github.com/retroenv/retroasm/lexer/token"
	"github.com/retroenv/retroasm/number"
	"github.com/retroenv/retroasm/parser/ast"
)

// Include ...
func Include(p arch.Parser) (ast.Node, error) {
	command := p.NextToken(1)

	next := p.NextToken(2)
	if next.Type.IsTerminator() {
		return nil, errMissingParameter
	}

	fileName := next.Value
	escapedFileName := strings.Contains(fileName, ".")
	if !escapedFileName {
		next3 := p.NextToken(3)
		next4 := p.NextToken(4)

		if next3.Type != token.Dot && !next4.Type.IsTerminator() {
			return nil, errMissingParameter
		}
		p.AdvanceReadPosition(4)
		fileName += "." + next4.Value
	} else {
		p.AdvanceReadPosition(2)
	}

	var start, size int

	next = p.NextToken(1)
	if next.Type == token.Comma {
		next = p.NextToken(2)
		i, err := number.Parse(next.Value)
		if err != nil {
			return nil, fmt.Errorf("parsing number '%s': %w", next.Value, err)
		}
		start = int(i)
		p.AdvanceReadPosition(2)
	}

	next = p.NextToken(1)
	if next.Type == token.Comma {
		next = p.NextToken(2)
		i, err := number.Parse(next.Value)
		if err != nil {
			return nil, fmt.Errorf("parsing number '%s': %w", next.Value, err)
		}
		size = int(i)
		p.AdvanceReadPosition(2)
	}

	binary := directiveBinaryIncludes.Contains(strings.ToLower(command.Value))

	return ast.NewInclude(fileName, binary, start, size), nil
}
