package ast

import (
	"math"

	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/number"
)

// ParseSymbolReference returns a normalized symbol and constant addend.
func ParseSymbolReference(exp *expression.Expression) (string, int64, bool) {
	if exp == nil {
		return "", 0, false
	}

	tokens := exp.Tokens()
	if len(tokens) == 1 && tokens[0].Type == token.Identifier {
		return tokens[0].Value, 0, true
	}
	if len(tokens) != 3 {
		return "", 0, false
	}

	if tokens[0].Type == token.Identifier && tokens[2].Type == token.Number {
		return parseTrailingAddend(tokens[0].Value, tokens[1].Type, tokens[2].Value)
	}
	if tokens[0].Type == token.Number && tokens[1].Type == token.Plus && tokens[2].Type == token.Identifier {
		addend, ok := parsePositiveAddend(tokens[0].Value)
		return tokens[2].Value, addend, ok
	}
	return "", 0, false
}

func parseTrailingAddend(symbol string, operator token.Type, value string) (string, int64, bool) {
	addend, err := number.Parse(value)
	if err != nil {
		return "", 0, false
	}

	switch operator {
	case token.Plus:
		if addend > math.MaxInt64 {
			return "", 0, false
		}
		return symbol, int64(addend), true
	case token.Minus:
		if addend > uint64(math.MaxInt64)+1 {
			return "", 0, false
		}
		if addend == uint64(math.MaxInt64)+1 {
			return symbol, math.MinInt64, true
		}
		return symbol, -int64(addend), true
	default:
		return "", 0, false
	}
}

func parsePositiveAddend(value string) (int64, bool) {
	addend, err := number.Parse(value)
	if err != nil || addend > math.MaxInt64 {
		return 0, false
	}
	return int64(addend), true
}
