package directives

import (
	"errors"
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

// MotorolaData parses DC.B, DC.W, and DC.L data definitions.
func MotorolaData(p arch.Parser) (ast.Node, error) {
	p.AdvanceReadPosition(1)
	return readMotorolaData(p, false)
}

func readMotorolaData(p arch.Parser, repeated bool) (ast.Node, error) {
	if p.NextToken(1).Type != token.Dot {
		return nil, errors.New("expected motorola data width suffix")
	}
	var width int
	switch strings.ToUpper(p.NextToken(2).Value) {
	case "B":
		width = 1
	case "W":
		width = 2
	case "L":
		width = 4
	default:
		return nil, fmt.Errorf("invalid motorola data width '%s'", p.NextToken(2).Value)
	}

	p.AdvanceReadPosition(2)
	if !repeated {
		values, err := readDataValueExpressions(p)
		if err != nil {
			return nil, fmt.Errorf("reading motorola data values: %w", err)
		}
		return constantData(p, width, values), nil
	}
	return readMotorolaBlock(p, width)
}

func readMotorolaBlock(p arch.Parser, width int) (ast.Node, error) {
	tokens, err := readDataTokens(p, true)
	if err != nil {
		return nil, fmt.Errorf("reading motorola data count: %w", err)
	}
	if len(tokens) == 0 {
		return nil, errMissingParameter
	}
	data := ast.NewData(ast.DataType, width)
	data.Fill = true
	data.Size = expression.New(tokens...)
	if p.NextToken(1).Type.IsTerminator() {
		p.AdvanceReadPosition(1)
		return data, nil
	}
	values, err := readDataValueExpressions(p)
	if err != nil {
		return nil, fmt.Errorf("reading motorola fill value: %w", err)
	}
	if len(values) != 1 {
		return nil, errors.New("expected one motorola fill value")
	}
	data.Values = values

	return data, nil
}
