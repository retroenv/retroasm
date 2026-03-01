package parser

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
)

var errMissingOperand = errors.New("missing operand")

// ParseIdentifier parses a Z80 instruction and resolves the matching instruction variant.
func ParseIdentifier(parser arch.Parser, mnemonic string, variants []*cpuz80.Instruction) (ast.Node, error) {
	operands, err := parseOperands(parser)
	if err != nil {
		return nil, fmt.Errorf("parsing operands: %w", err)
	}

	resolved, err := resolveInstruction(variants, operands)
	if err != nil {
		return nil, fmt.Errorf("resolving instruction '%s': %w", mnemonic, err)
	}

	argument := ast.NewInstructionArgument(*resolved)
	return ast.NewInstruction(mnemonic, int(resolved.Addressing), argument, nil), nil
}

func parseOperands(parser arch.Parser) ([]rawOperand, error) {
	next := parser.NextToken(1)
	if next.Type.IsTerminator() {
		return nil, nil
	}

	parser.AdvanceReadPosition(1)

	operand1, err := parseOperand(parser)
	if err != nil {
		return nil, err
	}

	if parser.NextToken(1).Type != token.Comma {
		return []rawOperand{operand1}, nil
	}

	parser.AdvanceReadPosition(2)
	operand2, err := parseOperand(parser)
	if err != nil {
		return nil, err
	}

	return []rawOperand{operand1, operand2}, nil
}

func parseOperand(parser arch.Parser) (rawOperand, error) {
	tok := parser.NextToken(0)

	switch tok.Type {
	case token.Number, token.Identifier:
		offsetOperand, matched, err := parseOffsetOperand(parser, tok)
		if err != nil {
			return rawOperand{}, err
		}
		if matched {
			return offsetOperand, nil
		}
		return rawOperand{token: tok}, nil
	case token.LeftParentheses:
		return parseParenthesizedOperand(parser)
	case token.EOF, token.EOL, token.Comment:
		return rawOperand{}, errMissingOperand
	default:
		return rawOperand{}, fmt.Errorf("unsupported operand token type %s", tok.Type)
	}
}

func parseParenthesizedOperand(parser arch.Parser) (rawOperand, error) {
	inner := parser.NextToken(1)
	if inner.Type.IsTerminator() {
		return rawOperand{}, errMissingOperand
	}

	switch inner.Type {
	case token.Identifier:
		return parseParenthesizedIdentifierOperand(parser, inner)
	case token.Number:
		return parseParenthesizedValueOperand(parser, inner)
	default:
		return rawOperand{}, fmt.Errorf("unsupported parenthesized operand token type %s", inner.Type)
	}
}

func parseParenthesizedIdentifierOperand(parser arch.Parser, identifier token.Token) (rawOperand, error) {
	next := parser.NextToken(2)
	switch next.Type {
	case token.RightParentheses:
		parser.AdvanceReadPosition(2)

		if indexedOperand, ok, err := parseEmbeddedIndexedIdentifier(identifier.Value); ok || err != nil {
			return indexedOperand, err
		}

		candidates := registerCandidatesForIndirectIdentifier(identifier.Value)
		if len(candidates) > 0 {
			return rawOperand{
				registerParams: candidates,
				parenthesized:  true,
			}, nil
		}

		return rawOperand{
			parenthesized: true,
			value:         ast.NewLabel(identifier.Value),
		}, nil

	case token.Plus, token.Minus:
		if _, ok := indexedIndirectRegister(identifier.Value); ok {
			return parseIndexedOperand(parser, identifier.Value, next.Type)
		}

		return parseParenthesizedOffsetOperand(parser, identifier, next.Type)

	default:
		return rawOperand{}, fmt.Errorf("unsupported parenthesized identifier form near '%s'", identifier.Value)
	}
}

func parseEmbeddedIndexedIdentifier(value string) (rawOperand, bool, error) {
	if !strings.Contains(value, "-") {
		return rawOperand{}, false, nil
	}

	parts := strings.SplitN(value, "-", 2)
	if len(parts) != 2 || parts[1] == "" {
		return rawOperand{}, false, fmt.Errorf("invalid indexed identifier '%s'", value)
	}

	registerParam, ok := indexedIndirectRegister(parts[0])
	if !ok {
		return rawOperand{}, false, nil
	}

	displacement, err := parseIndexedDisplacement(token.Token{
		Type:  token.Number,
		Value: parts[1],
	}, token.Minus)
	if err != nil {
		return rawOperand{}, false, err
	}

	return rawOperand{
		displacement:   displacement,
		parenthesized:  true,
		registerParams: []cpuz80.RegisterParam{registerParam},
	}, true, nil
}

func parseParenthesizedValueOperand(parser arch.Parser, valueToken token.Token) (rawOperand, error) {
	next := parser.NextToken(2)

	switch next.Type {
	case token.RightParentheses:
		parser.AdvanceReadPosition(2)

		value, ok, err := parseValueOperand(valueToken)
		if err != nil {
			return rawOperand{}, err
		}
		if !ok {
			return rawOperand{}, fmt.Errorf("unsupported parenthesized value '%s'", valueToken.Value)
		}

		return rawOperand{
			parenthesized: true,
			value:         value,
		}, nil

	case token.Plus, token.Minus:
		return parseParenthesizedOffsetOperand(parser, valueToken, next.Type)

	default:
		return rawOperand{}, errors.New("missing closing parenthesis")
	}
}

func parseIndexedOperand(parser arch.Parser, base string, operator token.Type) (rawOperand, error) {
	registerParam, ok := indexedIndirectRegister(base)
	if !ok {
		return rawOperand{}, fmt.Errorf("unsupported indexed base register '%s'", base)
	}

	displacementToken := parser.NextToken(3)
	if displacementToken.Type != token.Number {
		return rawOperand{}, fmt.Errorf("expected numeric displacement in indexed operand, got %s", displacementToken.Type)
	}
	if parser.NextToken(4).Type != token.RightParentheses {
		return rawOperand{}, errors.New("missing closing parenthesis")
	}

	displacement, err := parseIndexedDisplacement(displacementToken, operator)
	if err != nil {
		return rawOperand{}, err
	}

	parser.AdvanceReadPosition(4)

	return rawOperand{
		displacement:   displacement,
		parenthesized:  true,
		registerParams: []cpuz80.RegisterParam{registerParam},
	}, nil
}

func parseOffsetOperand(parser arch.Parser, base token.Token) (rawOperand, bool, error) {
	operator := parser.NextToken(1).Type
	if operator != token.Plus && operator != token.Minus {
		return rawOperand{}, false, nil
	}

	offsetToken := parser.NextToken(2)
	if offsetToken.Type != token.Number {
		return rawOperand{}, false, fmt.Errorf("expected numeric offset after '%s', got %s", operator, offsetToken.Type)
	}

	value, err := parseOffsetValue(base, operator, offsetToken)
	if err != nil {
		return rawOperand{}, false, err
	}

	parser.AdvanceReadPosition(2)
	return rawOperand{value: value}, true, nil
}

func parseParenthesizedOffsetOperand(parser arch.Parser, base token.Token, operator token.Type) (rawOperand, error) {
	offsetToken := parser.NextToken(3)
	if offsetToken.Type != token.Number {
		return rawOperand{}, fmt.Errorf("expected numeric offset in parenthesized operand, got %s", offsetToken.Type)
	}
	if parser.NextToken(4).Type != token.RightParentheses {
		return rawOperand{}, errors.New("missing closing parenthesis")
	}

	value, err := parseOffsetValue(base, operator, offsetToken)
	if err != nil {
		return rawOperand{}, err
	}

	parser.AdvanceReadPosition(4)
	return rawOperand{
		parenthesized: true,
		value:         value,
	}, nil
}

func parseOffsetValue(base token.Token, operator token.Type, offsetToken token.Token) (ast.Node, error) {
	offset, err := parseNumericToken(offsetToken)
	if err != nil {
		return nil, fmt.Errorf("parsing offset value '%s': %w", offsetToken.Value, err)
	}

	switch base.Type {
	case token.Identifier:
		if offset == 0 {
			return ast.NewLabel(base.Value), nil
		}

		if operator == token.Plus {
			return ast.NewLabel(fmt.Sprintf("%s+%d", base.Value, offset)), nil
		}
		return ast.NewLabel(fmt.Sprintf("%s-%d", base.Value, offset)), nil

	case token.Number:
		baseValue, err := parseNumericToken(base)
		if err != nil {
			return nil, fmt.Errorf("parsing base value '%s': %w", base.Value, err)
		}

		switch operator {
		case token.Plus:
			if baseValue > math.MaxUint64-offset {
				return nil, fmt.Errorf("numeric offset overflow: %d + %d", baseValue, offset)
			}
			return ast.NewNumber(baseValue + offset), nil
		case token.Minus:
			if baseValue < offset {
				return nil, fmt.Errorf("numeric offset underflow: %d - %d", baseValue, offset)
			}
			return ast.NewNumber(baseValue - offset), nil
		default:
			return nil, fmt.Errorf("unsupported offset operator %s", operator)
		}

	default:
		return nil, fmt.Errorf("unsupported offset base token type %s", base.Type)
	}
}

func parseNumericToken(tok token.Token) (uint64, error) {
	valueNode, ok, err := parseValueOperand(tok)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("unsupported numeric token '%s'", tok.Value)
	}

	numberValue, ok := valueNode.(ast.Number)
	if !ok {
		return 0, fmt.Errorf("unsupported numeric token type %T", valueNode)
	}

	return numberValue.Value, nil
}

func parseIndexedDisplacement(displacement token.Token, operator token.Type) (ast.Node, error) {
	value, err := parseNumericToken(displacement)
	if err != nil {
		return nil, fmt.Errorf("invalid indexed displacement '%s': %w", displacement.Value, err)
	}
	if value > 0xFF {
		return nil, fmt.Errorf("indexed displacement %d exceeds byte", value)
	}

	if operator == token.Plus {
		return ast.NewNumber(value), nil
	}

	if value > 0x80 {
		return nil, fmt.Errorf("indexed negative displacement %d exceeds signed byte range", value)
	}
	if value == 0 {
		return ast.NewNumber(0), nil
	}

	return ast.NewNumber(0x100 - value), nil
}
