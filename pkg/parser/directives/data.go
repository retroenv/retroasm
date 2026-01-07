package directives

import (
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

var dataByteWidth = map[string]int{
	"align": 1,
	"byt":   1,
	"byte":  1,
	"db":    1,
	"dcb":   1,
	"dcw":   1,
	"dsb":   1,
	"dsw":   2,
	"dw":    2,
	"pad":   1, // padding
	"word":  2,
}

// Data ...
func Data(p arch.Parser) (ast.Node, error) {
	if p.NextToken(2).Type.IsTerminator() {
		return nil, errMissingParameter
	}

	p.AdvanceReadPosition(1)
	typ := p.NextToken(0)
	typName := strings.ToLower(typ.Value)
	width, ok := dataByteWidth[typName]
	if !ok {
		return nil, fmt.Errorf("data width for type '%s' not found", typ.Value)
	}

	data := ast.NewData(ast.DataType, width)

	tokens, err := readDataTokens(p, false)
	if err != nil {
		return nil, fmt.Errorf("reading data tokens: %w", err)
	}
	data.Values = expression.New(tokens...)

	return data, nil
}

// DataStorage ...
func DataStorage(p arch.Parser) (ast.Node, error) {
	return readDataStorageTokens(p)
}

// Padding ...
func Padding(p arch.Parser) (ast.Node, error) {
	data, err := readDataStorageTokens(p)
	if err != nil {
		return nil, err
	}

	return addSizeProgramCounterReference(data)
}

// Align ...
func Align(p arch.Parser) (ast.Node, error) {
	data, err := readDataStorageTokens(p)
	if err != nil {
		return nil, err
	}

	// calculate size-$%size to get size until align count
	// TODO is data generated if address is already aligned?
	// if not the calculation should be (size-$%size)%size
	programCounter := token.Token{
		Type:  token.Number,
		Value: expression.ProgramCounterReference,
	}
	percent := token.Token{
		Type: token.Percent,
	}
	minus := token.Token{
		Type: token.Minus,
	}
	tokens := data.Size.Tokens()

	data.Size = expression.New(tokens...)
	data.Size.AddTokens(minus, programCounter, percent)
	data.Size.AddTokens(tokens...)
	return data, nil
}

func addSizeProgramCounterReference(data ast.Data) (ast.Node, error) {
	minus := token.Token{
		Type: token.Minus,
	}
	programCounter := token.Token{
		Type:  token.Number,
		Value: expression.ProgramCounterReference,
	}
	data.Size.AddTokens(minus, programCounter)
	return data, nil
}

func readDataStorageTokens(p arch.Parser) (ast.Data, error) {
	if p.NextToken(2).Type.IsTerminator() {
		return ast.Data{}, errMissingParameter
	}

	p.AdvanceReadPosition(1)
	typ := p.NextToken(0)
	typName := strings.ToLower(typ.Value)
	width, ok := dataByteWidth[typName]
	if !ok {
		return ast.Data{}, fmt.Errorf("data width for type '%s' not found", typ.Value)
	}

	data := ast.NewData(ast.DataType, width)
	data.Fill = true

	tokens, err := readDataTokens(p, true)
	if err != nil {
		return ast.Data{}, fmt.Errorf("reading data size tokens: %w", err)
	}
	data.Size = expression.New(tokens...)

	// fill value is optional, defaults to 0
	if p.NextToken(1).Type.IsTerminator() {
		p.AdvanceReadPosition(1)
		return data, nil
	}

	tokens, err = readDataTokens(p, false)
	if err != nil {
		return ast.Data{}, fmt.Errorf("reading data tokens: %w", err)
	}
	data.Values = expression.New(tokens...)
	return data, nil
}

// readDataTokens reads size or data tokens of a data directive.
// For size tokens a comma indicates the end of the tokens,
// for data values it acts as a separator.
func readDataTokens(p arch.Parser, returnOnComma bool) ([]token.Token, error) {
	var tokens []token.Token

	// read all tokens until the terminator
	for {
		p.AdvanceReadPosition(1)
		tok := p.NextToken(0)

		switch {
		case tok.Type == token.Number,
			tok.Type == token.Identifier,
			tok.Type.IsOperator():
			tokens = append(tokens, tok)

		case tok.Type == token.Comma:
			if returnOnComma {
				return tokens, nil
			}

		case tok.Type == token.Assign:
			if len(tokens) == 0 {
				tokens = append(tokens, tok)
				break
			}

			// check if the previous token is any of < > or =
			// to update the previous token type
			lastPos := len(tokens) - 1
			previous := tokens[lastPos]
			switch previous.Type {
			case token.Lt:
				tokens[lastPos].Type = token.LtE
			case token.Gt:
				tokens[lastPos].Type = token.GtE
			case token.Assign:
				tokens[lastPos].Type = token.Equals
			default:
				tokens = append(tokens, tok)
			}

		default:
			return nil, fmt.Errorf("unexpected token type found: '%s'", tok.Type.String())
		}

		tok = p.NextToken(1)
		if tok.Type.IsTerminator() {
			return tokens, nil
		}
	}
}
