// Package parser processes an input stream and parses its token to produce
// an abstract syntax tree (AST) as output.
package parser

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/retroenv/assembler/arch"
	"github.com/retroenv/assembler/expression"
	"github.com/retroenv/assembler/lexer"
	"github.com/retroenv/assembler/lexer/token"
	"github.com/retroenv/assembler/number"
	"github.com/retroenv/assembler/parser/ast"
	"github.com/retroenv/assembler/parser/directives"
	. "github.com/retroenv/retrogolib/addressing"
)

var errMissingParameter = errors.New("missing parameter")

// Parser is the input stream parser.
type Parser struct {
	arch          arch.Architecture
	lexer         *lexer.Lexer
	program       []token.Token
	readPosition  int
	programLength int
}

// New returns a new Parser that uses a lexer for the given reader.
func New(arch arch.Architecture, reader io.Reader) *Parser {
	lexerCfg := lexer.Config{
		CommentPrefixes: []string{"//", ";"},
		DecimalPrefix:   '#',
	}
	return &Parser{
		arch:  arch,
		lexer: lexer.New(lexerCfg, reader),
	}
}

// NewWithTokens returns a new Parser that processes the lexed tokens.
func NewWithTokens(arch arch.Architecture, tokens []token.Token) *Parser {
	return &Parser{
		arch:          arch,
		program:       tokens,
		programLength: len(tokens),
	}
}

// Read all tokens of the lexer.
func (p *Parser) Read() error {
	if err := p.parseTokens(); err != nil {
		return fmt.Errorf("parsing tokens: %w", err)
	}
	return nil
}

// NextToken returns the current or a following token with the given offset from current token parse position.
// If the offset exceeds the available tokens, a token of type EOF is returned.
func (p *Parser) NextToken(offset int) token.Token {
	if p.readPosition+offset >= p.programLength {
		return token.Token{
			Type: token.EOF,
		}
	}
	return p.program[p.readPosition+offset]
}

// AdvanceReadPosition advances the token read position.
func (p *Parser) AdvanceReadPosition(offset int) {
	p.readPosition += offset
}

// Arch returns the architecture set for the parser.
func (p *Parser) Arch() arch.Architecture {
	return p.arch
}

// TokensToAstNodes converts tokens previously read or passed to the constructor to AST nodes.
func (p *Parser) TokensToAstNodes() ([]ast.Node, error) {
	var (
		err          error
		nodes        []ast.Node
		previousNode ast.Node
	)

	for p.readPosition < p.programLength {
		tok := p.program[p.readPosition]
		var entry ast.Node

		switch tok.Type {
		case token.Dot:
			entry, err = p.parseDot()

		case token.Identifier:
			entry, err = p.parseIdentifier(tok)

		case token.Number:
			entry, err = p.parseNumber(tok)

		case token.Lt:
			// set read position back since dot directive handler expect a directive
			p.AdvanceReadPosition(-1)
			entry, err = directives.AddrLow(p)

		case token.Gt:
			// set read position back since dot directive handler expect a directive
			p.AdvanceReadPosition(-1)
			entry, err = directives.AddrHigh(p)

		case token.Comment:
			entry = p.parseComment(tok, previousNode)

		case token.EOL:

		default:
			return nil, fmt.Errorf("unexpected token of type %s found at line %d column %d",
				tok.Type.String(), tok.Position.Line, tok.Position.Column)
		}

		if err != nil {
			return nil, fmt.Errorf("parser error for token '%s' of type %s found at line %d column %d: %w",
				tok.Value, tok.Type.String(), tok.Position.Line, tok.Position.Column, err)
		}
		if entry != nil {
			nodes = append(nodes, entry)
		}
		previousNode = entry
		p.readPosition++
	}

	return nodes, nil
}

func (p *Parser) parseTokens() error {
	for {
		tok, err := p.lexer.NextToken()
		if err != nil {
			return fmt.Errorf("reading next token: %w", err)
		}
		if tok.Type == token.Illegal {
			return fmt.Errorf("illegal token '%s' found at line %d column %d",
				tok.Value, tok.Position.Line, tok.Position.Column)
		}
		if tok.Type == token.EOF {
			break
		}
		p.program = append(p.program, tok)
	}

	p.programLength = len(p.program)
	return nil
}

// parseComment returns a new comment AST node or attaches the comment to the previous node if the comment is on the
// same line.
func (p *Parser) parseComment(tok token.Token, previousNode ast.Node) ast.Node {
	message := strings.TrimSpace(tok.Value)

	if previousNode != nil {
		previousNode.SetComment(message)
		return nil
	}

	return &ast.Comment{
		Message: message,
	}
}

func (p *Parser) parseDot() (ast.Node, error) {
	next := p.NextToken(1)
	if next.Type.IsTerminator() {
		return nil, errMissingParameter
	}
	directive := strings.ToLower(next.Value)
	handler, ok := directives.Handlers[directive]
	if !ok {
		return nil, fmt.Errorf("unsupported directive '%s'", next.Value)
	}

	return handler(p)
}

func (p *Parser) parseIdentifier(tok token.Token) (ast.Node, error) {
	next := p.NextToken(1)
	next2 := p.NextToken(2)

	switch {
	case next.Type == token.Colon: // "identifier:"
		p.readPosition++
		return ast.NewLabel(tok.Value), nil

	case next.Type == token.Assign: // "identifier = number"
		return p.parseAlias(tok, next)

		// nesasm identifier .rs number
	case next.Type == token.Dot &&
		next2.Type == token.Identifier &&
		strings.ToLower(next2.Value) == "rs":
		return p.parseNesAsmVariable(tok)
	}

	instructionName := strings.ToLower(tok.Value)
	ins, ok := p.arch.Instructions[instructionName]
	if !ok {
		n, err := p.parseAlias(tok, next)
		if err != nil {
			if errors.Is(err, errUnsupportedIdentifier) {
				return p.createIdentifier(tok)
			}
			return n, err
		}
		return n, nil
	}

	if len(ins.Addressing) == 1 && ins.HasAddressing(ImpliedAddressing) {
		return ast.NewInstruction(ins.Name, ImpliedAddressing, nil, nil), nil
	}

	node, err := p.parseInstruction(ins)
	if err != nil {
		return nil, fmt.Errorf("parsing instruction %s: %w", ins.Name, err)
	}
	return node, nil
}

func (p *Parser) parseNesAsmVariable(tok token.Token) (ast.Node, error) {
	value := p.NextToken(3)
	if value.Type != token.Number {
		return nil, fmt.Errorf("unsupported offset counter value type %s", value.Type)
	}

	i, err := number.Parse(value.Value)
	if err != nil {
		return nil, fmt.Errorf("parsing number '%s': %w", value.Value, err)
	}

	p.readPosition += 3
	v := ast.NewVariable(tok.Value, int(i))
	v.UseOffsetCounter = true
	return v, nil
}

func (p *Parser) parseNumber(tok token.Token) (ast.Node, error) {
	if tok.Value != expression.ProgramCounterReference {
		return nil, fmt.Errorf("unexpected token of type %s found at line %d column %d",
			tok.Type.String(), tok.Position.Line, tok.Position.Column)
	}

	node, err := directives.Base(p)
	if err != nil {
		return nil, fmt.Errorf("processing program counter assignment: %w", err)
	}
	return node, nil
}

func (p *Parser) createIdentifier(tok token.Token) (ast.Node, error) {
	i := ast.NewIdentifier(tok.Value)
	p.AdvanceReadPosition(1)

	for end := false; !end; {
		tok = p.NextToken(0)

		switch tok.Type {
		case token.Identifier, token.Number:
			i.Arguments = append(i.Arguments, tok)
		case token.Comma:
		default:
			end = true
			continue
		}

		p.AdvanceReadPosition(1)
	}

	return i, nil
}
