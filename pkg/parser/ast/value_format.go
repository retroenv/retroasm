package ast

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/number"
)

var ErrUnsupportedValueFormat = errors.New("unsupported AST value format")

// ValueFormatOptions controls target-neutral scalar and expression spelling.
type ValueFormatOptions struct {
	Decimal          bool
	MinimumHexDigits int
}

// FormatValue returns deterministic assembly spelling for an AST scalar or expression.
func FormatValue(value Node, options ValueFormatOptions) (string, error) {
	switch typed := value.(type) {
	case Number:
		return formatNumberValue(typed.Value, options), nil
	case *Number:
		if typed != nil {
			return formatNumberValue(typed.Value, options), nil
		}
	case Label:
		return typed.Name, nil
	case *Label:
		if typed != nil {
			return typed.Name, nil
		}
	case Identifier:
		return typed.Name, nil
	case *Identifier:
		if typed != nil {
			return typed.Name, nil
		}
	case Expression:
		return formatExpressionValue(typed.Value)
	case *Expression:
		if typed != nil {
			return formatExpressionValue(typed.Value)
		}
	}
	return "", fmt.Errorf("%w: %T", ErrUnsupportedValueFormat, value)
}

func formatNumberValue(value uint64, options ValueFormatOptions) string {
	if options.Decimal {
		return strconv.FormatUint(value, 10)
	}
	if options.MinimumHexDigits > 0 {
		return fmt.Sprintf("0x%0*X", options.MinimumHexDigits, value)
	}
	return fmt.Sprintf("0x%X", value)
}

func formatExpressionValue(value *expression.Expression) (string, error) {
	if value == nil {
		return "", ErrUnsupportedValueFormat
	}

	var builder strings.Builder
	for _, expressionToken := range value.Tokens() {
		tokenValue := expressionToken.Value
		if expressionToken.Type == token.Number && tokenValue != "$" {
			parsed, err := number.Parse(tokenValue)
			if err != nil {
				return "", fmt.Errorf("%w: invalid number %q: %w", ErrUnsupportedValueFormat, tokenValue, err)
			}
			tokenValue = fmt.Sprintf("0x%X", parsed)
		}
		if tokenValue == "" {
			tokenValue = expressionToken.Type.String()
		}
		builder.WriteString(tokenValue)
	}
	if builder.Len() == 0 {
		return "", ErrUnsupportedValueFormat
	}
	return builder.String(), nil
}
