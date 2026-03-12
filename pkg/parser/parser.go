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
	"github.com/retroenv/retroasm/pkg/assembler/config"
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
	compatMode    config.CompatibilityMode
	handlers      map[string]directives.Handler
	lexer         *lexer.Lexer
	program       []token.Token
	readPosition  int
	programLength int

	// anonymous label tracking for +/- labels
	anonForwardCount  int // total forward labels seen so far
	anonBackwardCount int // total backward labels seen so far

	// @local label scoping: tracks the last non-local label name
	lastNonLocalLabel string

	// ca65-style unnamed label tracking for : / :- / :+ labels
	unnamedLabelCount int
}

// New returns a new Parser that uses a lexer for the given reader.
func New[T any](arch arch.Architecture[T], reader io.Reader, mode config.CompatibilityMode) *Parser[T] {
	lexerCfg := lexer.Config{
		CommentPrefixes: []string{"//", ";"},
		DecimalPrefix:   '#',
	}
	return &Parser[T]{
		arch:       arch,
		compatMode: mode,
		handlers:   directives.BuildHandlers(mode),
		lexer:      lexer.New(lexerCfg, reader),
	}
}

// NewWithTokens returns a new Parser that processes the lexed tokens.
func NewWithTokens[T any](arch arch.Architecture[T], tokens []token.Token, mode config.CompatibilityMode) *Parser[T] {
	return &Parser[T]{
		arch:          arch,
		compatMode:    mode,
		handlers:      directives.BuildHandlers(mode),
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

// ScopeLocalLabel applies @local label scoping to a name if applicable.
func (p *Parser[T]) ScopeLocalLabel(name string) string {
	return p.scopeLocalLabel(name)
}

// ResolveDotLocalLabel returns the scoped name for a NESASM dot-prefixed local label.
func (p *Parser[T]) ResolveDotLocalLabel(name string) string {
	if !p.compatMode.DotLocalLabels() {
		return ""
	}
	return p.scopeDotLocalLabel("." + name)
}

// ResolveUnnamedLabel returns the synthetic label name for a ca65-style unnamed label reference.
func (p *Parser[T]) ResolveUnnamedLabel(forward bool, level int) string {
	if forward {
		return fmt.Sprintf("__unnamed_%d", p.unnamedLabelCount+level)
	}
	return fmt.Sprintf("__unnamed_%d", p.unnamedLabelCount-level+1)
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
		nodes        = make([]ast.Node, 0, p.programLength/2) // Pre-allocate with estimated capacity
		previousNode ast.Node
	)

	for p.readPosition < p.programLength {
		tok := p.program[p.readPosition]
		entry, err := p.parseToken(tok, previousNode)

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

//nolint:cyclop // type switch with one case per token type
func (p *Parser[T]) parseToken(tok token.Token, previousNode ast.Node) (ast.Node, error) {
	switch tok.Type {
	case token.Dot:
		return p.parseDot()

	case token.Identifier:
		return p.parseIdentifier(tok)

	case token.Number:
		return p.parseNumber(tok)

	case token.Lt:
		// set read position back since dot directive handler expect a directive
		p.AdvanceReadPosition(-1)
		return directives.AddrLow(p) //nolint:wrapcheck // thin delegation to sub-package

	case token.Gt:
		// set read position back since dot directive handler expect a directive
		p.AdvanceReadPosition(-1)
		return directives.AddrHigh(p) //nolint:wrapcheck // thin delegation to sub-package

	case token.Plus:
		if p.compatMode.AnonymousLabels() {
			return p.parseAnonymousLabel(true), nil
		}

	case token.Minus:
		if p.compatMode.AnonymousLabels() {
			return p.parseAnonymousLabel(false), nil
		}

	case token.Colon:
		if p.compatMode.UnnamedLabels() {
			return p.parseUnnamedLabel(), nil
		}

	case token.Asterisk:
		if p.compatMode.AsteriskProgramCounter() {
			return p.parseAsteriskPC()
		}

	case token.Comment:
		return p.parseComment(tok, previousNode), nil

	case token.EOL:
		return nil, nil //nolint:nilnil // EOL tokens produce no AST node
	}

	return nil, fmt.Errorf("unexpected token of type %s found at line %d column %d",
		tok.Type.String(), tok.Position.Line, tok.Position.Column)
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
	handler, ok := p.handlers[directive]
	if !ok {
		if p.compatMode.DotLocalLabels() {
			return p.parseDotLocalLabel(next)
		}
		return nil, fmt.Errorf("unsupported directive '%s'", next.Value)
	}

	return handler(p)
}

// parseDotLocalLabel handles NESASM-style dot-prefixed local labels (.label).
func (p *Parser[T]) parseDotLocalLabel(nameTok token.Token) (ast.Node, error) {
	name := "." + nameTok.Value
	p.readPosition++ // advance past the label name

	// Check for optional colon after the label name
	next := p.NextToken(1)
	if next.Type == token.Colon {
		p.readPosition++
	}

	scopedName := p.scopeDotLocalLabel(name)
	return ast.NewLabel(scopedName), nil
}

// scopeDotLocalLabel applies NESASM dot-local label scoping. If the name starts with '.'
// and there is a current non-local label scope, the name is prefixed to create a unique scoped name.
func (p *Parser[T]) scopeDotLocalLabel(name string) string {
	if p.lastNonLocalLabel == "" {
		return name
	}
	return p.lastNonLocalLabel + name
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
	case next.Type == token.Colon && !p.isUnnamedLabelRef(next2): // "identifier:"
		p.readPosition++
		name := p.scopeLocalLabel(tok.Value)
		p.updateLabelScope(tok.Value)
		return ast.NewLabel(name), nil

	case next.Type == token.Assign: // "identifier = number"
		return p.parseAlias(tok, next)

	case next.Type == token.Dot && next2.Type == token.Identifier && p.isDotIdentifierKeyword(next2):
		return p.parseDotIdentifier(tok, next2)
	}

	instructionName := strings.ToLower(tok.Value)
	ins, ok := p.arch.Instruction(instructionName)
	if !ok {
		// In colon-optional modes, an identifier at start of line that isn't an instruction
		// or directive may be a label without a trailing colon.
		if p.compatMode.ColonOptionalLabels() && p.isColonOptionalLabel(tok, next) {
			name := p.scopeLocalLabel(tok.Value)
			p.updateLabelScope(tok.Value)
			return ast.NewLabel(name), nil
		}

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

// isDotIdentifierKeyword checks if the token after a dot is a recognized keyword
// for "name .keyword" patterns (equ, rs, macro).
func (p *Parser[T]) isDotIdentifierKeyword(keyword token.Token) bool {
	switch strings.ToLower(keyword.Value) {
	case "equ", "rs":
		return true
	case "macro":
		return p.compatMode.NesasmMacroSyntax()
	}
	return false
}

// parseDotIdentifier handles "name .keyword" patterns:
//   - name .equ value (x816-style alias)
//   - name .rs number (NESASM variable)
//   - name .macro (NESASM macro definition)
func (p *Parser[T]) parseDotIdentifier(tok token.Token, keyword token.Token) (ast.Node, error) {
	switch strings.ToLower(keyword.Value) {
	case "equ":
		p.readPosition++ // skip the dot, so parseAlias sees "equ" as next
		return p.parseAlias(tok, keyword)

	case "rs":
		return p.parseNesAsmVariable(tok)

	case "macro":
		if p.compatMode.NesasmMacroSyntax() {
			return p.parseNesAsmMacro(tok)
		}
	}

	return nil, fmt.Errorf("unsupported dot-identifier '.%s'", keyword.Value)
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

// parseNesAsmMacro handles NESASM-style macro definition where name comes before .macro:
// name .macro
//
//nolint:cyclop // sequential checks for macro termination
func (p *Parser[T]) parseNesAsmMacro(nameTok token.Token) (ast.Node, error) {
	// Skip past: name . macro
	p.readPosition += 3
	m := ast.NewMacro(nameTok.Value)

	// NESASM macros don't have named parameters — they use \1-\9.
	// Read all macro tokens until .endm
	for end := false; !end; {
		tok := p.NextToken(0)
		p.AdvanceReadPosition(1)

		switch tok.Type {
		case token.EOF:
			end = true
			continue

		case token.Identifier:
			if strings.ToUpper(tok.Value) == "ENDM" {
				end = true
				continue
			}

		case token.Dot:
			// Check for .endm
			next := p.NextToken(0)
			if next.Type == token.Identifier {
				name := strings.ToLower(next.Value)
				if name == "endm" || name == "endmacro" {
					p.AdvanceReadPosition(1)
					end = true
					continue
				}
			}
		}

		m.Token = append(m.Token, tok)
	}

	return m, nil
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

// parseAnonymousLabel handles +/- anonymous label definitions.
// forward=true means + labels, forward=false means - labels.
// Consecutive +/- tokens increase the nesting level.
func (p *Parser[T]) parseAnonymousLabel(forward bool) ast.Node {
	level := 1
	for p.readPosition+level < p.programLength {
		next := p.program[p.readPosition+level]
		if (forward && next.Type == token.Plus) || (!forward && next.Type == token.Minus) {
			level++
		} else {
			break
		}
	}
	// advance past the extra +/- tokens (first one is consumed by the main loop)
	p.readPosition += level - 1

	var name string
	if forward {
		p.anonForwardCount++
		name = fmt.Sprintf("__anon_fwd_%d_%d", level, p.anonForwardCount)
	} else {
		p.anonBackwardCount++
		name = fmt.Sprintf("__anon_bwd_%d_%d", level, p.anonBackwardCount)
	}

	return ast.NewLabel(name)
}

// parseUnnamedLabel handles ca65-style unnamed label definitions (:).
// Each unnamed label gets a unique synthetic name based on the counter.
func (p *Parser[T]) parseUnnamedLabel() ast.Node {
	p.unnamedLabelCount++
	name := fmt.Sprintf("__unnamed_%d", p.unnamedLabelCount)
	return ast.NewLabel(name)
}

// parseAsteriskPC handles * as program counter assignment (e.g., * = $8000).
func (p *Parser[T]) parseAsteriskPC() (ast.Node, error) {
	next := p.NextToken(1)
	if next.Type == token.Assign {
		// * = value — program counter assignment, delegate to Base handler
		p.AdvanceReadPosition(1) // skip the =
		addressTokens, err := directives.ReadDataTokensExported(p, true)
		if err != nil {
			return nil, fmt.Errorf("reading program counter value: %w", err)
		}
		return ast.NewBase(addressTokens), nil
	}

	return nil, fmt.Errorf("unexpected token after '*' at line %d column %d",
		next.Position.Line, next.Position.Column)
}

// isColonOptionalLabel checks if the current identifier should be treated as a label
// without a trailing colon. This is true when the next token is an instruction,
// directive, EOL, or EOF, and the identifier is not a known directive.
func (p *Parser[T]) isColonOptionalLabel(tok, next token.Token) bool {
	// Must be at start of line (column 1 in lexer's 1-based column tracking)
	if tok.Position.Column > 1 {
		return false
	}

	// Check if it's a directive name (don't treat directives as labels)
	directive := strings.ToLower(tok.Value)
	if _, ok := p.handlers[directive]; ok {
		return false
	}

	// If next token is EOL/EOF, it's a label
	if next.Type.IsTerminator() {
		return true
	}

	// If next token is an instruction mnemonic, this is a label before an instruction
	if next.Type == token.Identifier {
		nextName := strings.ToLower(next.Value)
		if _, ok := p.arch.Instruction(nextName); ok {
			return true
		}
		// Also check if next is a directive name
		if _, ok := p.handlers[nextName]; ok {
			return true
		}
	}

	// If next is a dot (directive), this is a label before a directive
	if next.Type == token.Dot {
		return true
	}

	return false
}

// isUnnamedLabelRef checks if the token following a colon indicates an unnamed label reference
// (ca65-style :+/:- syntax) rather than a label definition colon.
func (p *Parser[T]) isUnnamedLabelRef(tokenAfterColon token.Token) bool {
	if !p.compatMode.UnnamedLabels() {
		return false
	}
	return tokenAfterColon.Type == token.Plus || tokenAfterColon.Type == token.Minus
}

// scopeLocalLabel applies @local label scoping. If the name starts with '@' and there
// is a current non-local label scope, the name is prefixed to create a unique scoped name.
func (p *Parser[T]) scopeLocalLabel(name string) string {
	if !p.compatMode.LocalLabelScoping() {
		return name
	}
	if len(name) == 0 || name[0] != '@' {
		return name
	}
	if p.lastNonLocalLabel == "" {
		return name
	}
	return p.lastNonLocalLabel + "." + name
}

// updateLabelScope tracks non-local labels for @local and dot-local label scoping.
func (p *Parser[T]) updateLabelScope(name string) {
	if !p.compatMode.LocalLabelScoping() && !p.compatMode.DotLocalLabels() {
		return
	}
	// Anonymous labels, @local labels, and .local labels don't update scope.
	if strings.HasPrefix(name, "__anon_") || strings.HasPrefix(name, "__unnamed_") {
		return
	}
	if len(name) > 0 && (name[0] == '@' || name[0] == '.') {
		return
	}
	p.lastNonLocalLabel = name
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
