package directives

import (
	"fmt"
	"strings"

	"github.com/retroenv/assembler/lexer/token"
	"github.com/retroenv/assembler/number"
	"github.com/retroenv/assembler/parser/ast"
)

// Include ...
func Include(p Parser) (ast.Node, error) {
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

	_, binary := directiveBinaryIncludes[strings.ToLower(command.Value)]

	return &ast.Include{
		Name:   fileName,
		Binary: binary,
		Start:  start,
		Size:   size,
	}, nil
}
