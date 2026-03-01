package parser

import (
	"errors"
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/arch/z80/profile"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpuz80 "github.com/retroenv/retrogolib/arch/cpu/z80"
)

var errMissingOperand = errors.New("missing operand")

// ParseIdentifier parses a Z80 instruction and resolves the matching instruction variant.
func ParseIdentifier(parser arch.Parser, mnemonic string, variants []*cpuz80.Instruction) (ast.Node, error) {
	return ParseIdentifierWithProfile(parser, mnemonic, variants, profile.Default)
}

// ParseIdentifierWithProfile parses a Z80 instruction and enforces the selected profile.
func ParseIdentifierWithProfile(
	parser arch.Parser,
	mnemonic string,
	variants []*cpuz80.Instruction,
	profileKind profile.Kind,
) (ast.Node, error) {

	operands, err := parseOperands(parser)
	if err != nil {
		return nil, fmt.Errorf("parsing operands: %w", err)
	}

	resolved, err := resolveInstruction(variants, operands)
	if err != nil {
		return nil, fmt.Errorf("resolving instruction '%s': %w", mnemonic, err)
	}

	if err := profile.ValidateInstruction(
		profileKind,
		resolved.Instruction,
		resolved.Addressing,
		resolved.RegisterParams,
	); err != nil {
		return nil, fmt.Errorf("validating profile '%s': %w", profileKind.String(), err)
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
		expressionOperand, matched, err := parseExpressionOperand(parser, tok)
		if err != nil {
			return rawOperand{}, err
		}
		if matched {
			return expressionOperand, nil
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

		return parseParenthesizedExpressionOperand(parser, identifier, next.Type)

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
		return parseParenthesizedExpressionOperand(parser, valueToken, next.Type)

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
	if displacementToken.Type == token.Number && parser.NextToken(4).Type == token.RightParentheses {
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

	displacement, consumed, err := parseIndexedExpressionDisplacement(parser, operator)
	if err != nil {
		return rawOperand{}, err
	}
	closingOffset := 3 + consumed
	if parser.NextToken(closingOffset).Type != token.RightParentheses {
		return rawOperand{}, errors.New("missing closing parenthesis")
	}
	parser.AdvanceReadPosition(closingOffset)

	return rawOperand{
		displacement:   displacement,
		parenthesized:  true,
		registerParams: []cpuz80.RegisterParam{registerParam},
	}, nil
}

func parseExpressionOperand(parser arch.Parser, base token.Token) (rawOperand, bool, error) {
	tokens, consumed, err := parseExpressionTokenList(parser, 1, token.Comma, true)
	if err != nil {
		return rawOperand{}, false, err
	}
	if consumed == 0 {
		return rawOperand{}, false, nil
	}

	parser.AdvanceReadPosition(consumed)
	return rawOperand{
		value: ast.NewExpression(append([]token.Token{base}, tokens...)...),
	}, true, nil
}

func parseParenthesizedExpressionOperand(parser arch.Parser, base token.Token, operator token.Type) (rawOperand, error) {
	tokens, consumed, err := parseExpressionTokenList(parser, 2, token.RightParentheses, true)
	if err != nil {
		return rawOperand{}, err
	}
	if consumed == 0 {
		return rawOperand{}, fmt.Errorf("expected numeric offset in parenthesized operand after '%s'", operator)
	}

	closingOffset := 2 + consumed
	if parser.NextToken(closingOffset).Type != token.RightParentheses {
		return rawOperand{}, errors.New("missing closing parenthesis")
	}

	parser.AdvanceReadPosition(closingOffset)
	return rawOperand{
		parenthesized: true,
		value:         ast.NewExpression(append([]token.Token{base}, tokens...)...),
	}, nil
}

func parseIndexedExpressionDisplacement(parser arch.Parser, operator token.Type) (ast.Node, int, error) {
	tokens, consumed, err := parseExpressionTokenList(parser, 3, token.RightParentheses, false)
	if err != nil {
		return nil, 0, err
	}
	if consumed == 0 {
		return nil, 0, fmt.Errorf("expected displacement expression after '%s'", operator)
	}

	if operator == token.Plus {
		return ast.NewExpression(tokens...), consumed, nil
	}

	negatedTokens := make([]token.Token, 0, len(tokens)+5)
	negatedTokens = append(negatedTokens, token.Token{Type: token.Number, Value: "0"})
	negatedTokens = append(negatedTokens, token.Token{Type: token.Minus})
	negatedTokens = append(negatedTokens, token.Token{Type: token.LeftParentheses})
	negatedTokens = append(negatedTokens, tokens...)
	negatedTokens = append(negatedTokens, token.Token{Type: token.RightParentheses})

	return ast.NewExpression(negatedTokens...), consumed, nil
}

func parseExpressionTokenList(
	parser arch.Parser,
	startOffset int,
	stopToken token.Type,
	requireLeadingOperator bool,
) ([]token.Token, int, error) {

	var tokens []token.Token

	for consumed := 0; ; consumed++ {
		tok := parser.NextToken(startOffset + consumed)

		if isExpressionStopToken(stopToken, tok.Type) {
			return finalizeExpressionTokens(tokens, consumed)
		}

		if err := validateExpressionToken(tokens, tok.Type, requireLeadingOperator); err != nil {
			return nil, 0, err
		}

		tokens = append(tokens, tok)
	}
}

func finalizeExpressionTokens(tokens []token.Token, consumed int) ([]token.Token, int, error) {
	if len(tokens) == 0 {
		return nil, 0, nil
	}

	lastToken := tokens[len(tokens)-1]
	if !isExpressionTokenEnd(lastToken.Type) {
		return nil, 0, fmt.Errorf("expected expression value after '%s'", lastToken.Type)
	}

	return tokens, consumed, nil
}

func isExpressionStopToken(stopToken, tokenType token.Type) bool {
	return tokenType == stopToken || tokenType.IsTerminator()
}

func validateExpressionToken(tokens []token.Token, tokenType token.Type, requireLeadingOperator bool) error {
	if len(tokens) == 0 {
		return validateExpressionTokenStart(tokenType, requireLeadingOperator)
	}

	previous := tokens[len(tokens)-1].Type
	if isExpressionTokenEnd(previous) && isExpressionTokenEnd(tokenType) {
		return fmt.Errorf("missing operator between expression tokens '%s' and '%s'", previous, tokenType)
	}

	if !isExpressionTokenAllowed(tokenType) {
		return fmt.Errorf("unsupported expression token type %s", tokenType)
	}

	return nil
}

func validateExpressionTokenStart(tokenType token.Type, requireLeadingOperator bool) error {
	if requireLeadingOperator && !isExpressionTokenStart(tokenType) {
		return fmt.Errorf("expected expression operator after base operand, got %s", tokenType)
	}

	if !requireLeadingOperator && !isExpressionValueStart(tokenType) {
		return fmt.Errorf("expected expression value token, got %s", tokenType)
	}

	if !isExpressionTokenAllowed(tokenType) {
		return fmt.Errorf("unsupported expression token type %s", tokenType)
	}

	return nil
}

func isExpressionTokenAllowed(tokenType token.Type) bool {
	return tokenType == token.Number ||
		tokenType == token.Identifier ||
		tokenType == token.LeftParentheses ||
		tokenType == token.RightParentheses ||
		tokenType.IsOperator()
}

func isExpressionTokenStart(tokenType token.Type) bool {
	return tokenType == token.Plus ||
		tokenType == token.Minus ||
		tokenType.IsOperator()
}

func isExpressionTokenEnd(tokenType token.Type) bool {
	return tokenType == token.Number ||
		tokenType == token.Identifier ||
		tokenType == token.RightParentheses
}

func isExpressionValueStart(tokenType token.Type) bool {
	return tokenType == token.Number ||
		tokenType == token.Identifier ||
		tokenType == token.LeftParentheses ||
		tokenType == token.Plus ||
		tokenType == token.Minus
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
