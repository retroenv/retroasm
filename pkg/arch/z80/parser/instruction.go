package parser

import (
	"errors"
	"fmt"

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

	operand1, err := parseOperandToken(parser.NextToken(0))
	if err != nil {
		return nil, err
	}

	if parser.NextToken(1).Type != token.Comma {
		return []rawOperand{operand1}, nil
	}

	parser.AdvanceReadPosition(2)
	operand2, err := parseOperandToken(parser.NextToken(0))
	if err != nil {
		return nil, err
	}

	return []rawOperand{operand1, operand2}, nil
}

func parseOperandToken(tok token.Token) (rawOperand, error) {
	switch tok.Type {
	case token.Number, token.Identifier:
		return rawOperand{token: tok}, nil
	case token.EOF, token.EOL, token.Comment:
		return rawOperand{}, errMissingOperand
	default:
		return rawOperand{}, fmt.Errorf("unsupported operand token type %s", tok.Type)
	}
}
