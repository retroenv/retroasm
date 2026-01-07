// Package parser processes an input stream and parses tokens to produce
// an abstract syntax tree (AST) as output.
//
// The parser supports two main workflows:
//   - Stream parsing: New() + Read() + TokensToAstNodes()
//   - Direct parsing: NewWithTokens() + TokensToAstNodes()
//
// The parser handles multiple assembly formats (asm6, ca65, nesasm) and supports:
//   - Instructions with various addressing modes
//   - Assembler directives (data, includes, conditionals, macros)
//   - Labels and aliases
//   - Comments and expressions
//
// Architecture-specific instruction parsing is delegated to the provided
// arch.Architecture implementation.
package parser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/number"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retroasm/pkg/parser/directives"
)

var errMissingParameter = errors.New("missing parameter")

// Parser is the input stream parser.
type Parser[T any] struct {
	arch          arch.Architecture[T]
	lexer         *lexer.Lexer
	program       []token.Token
	readPosition  int
	programLength int
}

// New returns a new Parser that uses a lexer for the given reader.
func New[T any](arch arch.Architecture[T], reader io.Reader) *Parser[T] {
	lexerCfg := lexer.Config{
		CommentPrefixes: []string{"//", ";"},
		DecimalPrefix:   '#',
	}
	return &Parser[T]{
		arch:  arch,
		lexer: lexer.New(lexerCfg, reader),
	}
}

// NewWithTokens returns a new Parser that processes the lexed tokens.
func NewWithTokens[T any](arch arch.Architecture[T], tokens []token.Token) *Parser[T] {
	return &Parser[T]{
		arch:          arch,
		program:       tokens,
		programLength: len(tokens),
	}
}

// Read all tokens of the lexer.
func (p *Parser[T]) Read(ctx context.Context) error {
	if err := p.parseTokens(ctx); err != nil {
		return fmt.Errorf("parsing tokens: %w", err)
	}
	return nil
}

// NextToken returns the current or a following token with the given offset from current token parse position.
// If the offset exceeds the available tokens, a token of type EOF is returned.
func (p *Parser[T]) NextToken(offset int) token.Token {
	if p.readPosition+offset >= p.programLength {
		return token.Token{
			Type: token.EOF,
		}
	}
	return p.program[p.readPosition+offset]
}

// AdvanceReadPosition advances the token read position.
func (p *Parser[T]) AdvanceReadPosition(offset int) {
	p.readPosition += offset
}

// AddressWidth returns the address width of the architecture.
func (p *Parser[T]) AddressWidth() int {
	return p.arch.AddressWidth()
}

// TokensToAstNodes converts tokens previously read or passed to the constructor to AST nodes.
//
// This is the core parsing method that processes tokens sequentially and creates
// corresponding AST nodes. It handles:
//   - Directives (starting with '.')
//   - Instructions (architecture-specific)
//   - Labels (identifier followed by ':')
//   - Aliases (identifier = value)
//   - Address operators ('<' for low byte, '>' for high byte)
//   - Comments (attached to previous node if on same line)
//
// The method maintains parsing state and provides detailed error context including
// line and column numbers for debugging.
func (p *Parser[T]) TokensToAstNodes() ([]ast.Node, error) {
	var (
		err          error
		nodes        = make([]ast.Node, 0, p.programLength/2) // Pre-allocate with estimated capacity
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

func (p *Parser[T]) parseTokens(ctx context.Context) error {
	for {
		// Check for cancellation in tokenization loop
		select {
		case <-ctx.Done():
			return fmt.Errorf("tokenization cancelled: %w", ctx.Err())
		default:
		}
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
func (p *Parser[T]) parseComment(tok token.Token, previousNode ast.Node) ast.Node {
	message := strings.TrimSpace(tok.Value)

	if previousNode != nil {
		previousNode.SetComment(message)
		return nil
	}

	return &ast.Comment{
		Message: message,
	}
}

func (p *Parser[T]) parseDot() (ast.Node, error) {
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

// parseIdentifier handles identifier tokens which can represent:
//   - Labels: "identifier:"
//   - Aliases: "identifier = value"
//   - NES ASM variables: "identifier .rs number"
//   - Instructions: delegated to architecture-specific parsing
//   - Generic identifiers: fallback for unknown patterns
func (p *Parser[T]) parseIdentifier(tok token.Token) (ast.Node, error) {
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
	ins, ok := p.arch.Instruction(instructionName)
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

	n, err := p.arch.ParseIdentifier(p, ins)
	if err != nil {
		return nil, fmt.Errorf("parsing identifier '%s': %w", tok.Value, err)
	}
	return n, nil
}

func (p *Parser[T]) parseNesAsmVariable(tok token.Token) (ast.Node, error) {
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

func (p *Parser[T]) parseNumber(tok token.Token) (ast.Node, error) {
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

func (p *Parser[T]) createIdentifier(tok token.Token) (ast.Node, error) {
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
