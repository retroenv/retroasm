// Package lexer implements a lexical analyzer that can be used for reading source and configuration files.
package lexer

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/retroenv/assembler/lexer/token"
)

// Lexer is the lexical analyzer.
type Lexer struct {
	cfg    Config
	reader *bufio.Reader
	pos    token.Position
}

// Config contains the lexer configuration.
type Config struct {
	CommentPrefixes []string // for example // and ; for assembly
	DecimalPrefix   rune     // for example # for assembly
}

// New returns a new lexer.
func New(cfg Config, reader io.Reader) *Lexer {
	return &Lexer{
		cfg:    cfg,
		reader: bufio.NewReader(reader),
		pos: token.Position{
			Line:   1,
			Column: 0,
		},
	}
}

// NextToken returns the next token or an EOF token if there are no more tokens available.
func (l *Lexer) NextToken() (token.Token, error) {
	for {
		r, _, err := l.reader.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return l.newToken(token.EOF), nil
			}

			return token.Token{}, fmt.Errorf("reading rune: %w", err)
		}

		l.pos.Column++

		t, ok, err := l.processRune(r)
		if err != nil {
			return token.Token{}, fmt.Errorf("processing rune: %w", err)
		}
		if ok {
			return t, nil
		}
	}
}

// processRune processes the read rune and returns the token and a flag
// whether the token is valid.
// nolint: cyclop
func (l *Lexer) processRune(r rune) (token.Token, bool, error) {
	switch {
	case r == '\n':
		l.pos.NextLine()
		return l.newToken(token.EOL), true, nil

	case unicode.IsSpace(r):
		return token.Token{}, false, nil

	case unicode.IsDigit(r) || r == '$' || r == '%' ||
		(l.cfg.DecimalPrefix != 0 && r == l.cfg.DecimalPrefix):
		tok, err := l.readNumber(r)
		if err != nil {
			return token.Token{}, false, err
		}
		return tok, true, nil

	case unicode.IsLetter(r) || r == '_' || r == '@':
		tok, err := l.readLiteral(r)
		if err != nil {
			return token.Token{}, false, err
		}
		return tok, true, nil

	case r == '"':
		tok, err := l.readString(r)
		if err != nil {
			return token.Token{}, false, err
		}
		return tok, true, nil
	}

	tok, commentFound, err := l.checkCommentPrefixes(r)
	if err != nil {
		return token.Token{}, false, err
	}
	if commentFound {
		return tok, true, nil
	}

	typ, err := token.NewType(r)
	if err != nil {
		return token.Token{}, false, fmt.Errorf("creating token type: %w", err)
	}

	return l.newToken(typ), true, nil
}

func (l *Lexer) checkCommentPrefixes(r rune) (token.Token, bool, error) {
	for _, prefix := range l.cfg.CommentPrefixes {
		if len(prefix) == 1 {
			if r == rune(prefix[0]) {
				tok, err := l.readComment(r)
				if err != nil {
					return token.Token{}, false, err
				}
				return tok, true, nil
			}
			continue
		}

		peek, err := l.reader.Peek(len(prefix) - 1)
		if err != nil {
			continue
		}

		s := string(r) + string(peek)
		if s != prefix {
			continue
		}

		_, _ = l.reader.Discard(len(prefix) - 1)
		tok, err := l.readComment(r)
		if err != nil {
			return token.Token{}, false, err
		}
		return tok, true, nil
	}
	return token.Token{}, false, nil
}

// readNumber reads a number token that can be of decimal, hex or binary type.
func (l *Lexer) readNumber(firstCharacter rune) (token.Token, error) {
	literal := &strings.Builder{}

	pos := l.pos
	isBinary := false
	hasPrefix := false // to make sure that only 1 prefix is processed

	if l.cfg.DecimalPrefix != 0 && firstCharacter == l.cfg.DecimalPrefix {
		literal.WriteRune(firstCharacter)
	} else {
		if err := l.reader.UnreadRune(); err != nil {
			return token.Token{}, fmt.Errorf("unreading rune: %w", err)
		}
		l.pos.Column--
	}

	for i := 0; ; i++ {
		r, _, err := l.reader.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return l.newTokenPosition(token.Number, literal.String(), pos), nil
			}

			return token.Token{}, fmt.Errorf("reading rune: %w", err)
		}

		l.pos.Column++

		characterHandled, characterValid := l.processNumberCharacter(firstCharacter, r, i, &isBinary, &hasPrefix, literal)
		if characterValid {
			continue
		}

		// if the token was not handled, rewind the reader
		if !characterHandled {
			l.pos.Column--
			if err := l.reader.UnreadRune(); err != nil {
				return token.Token{}, fmt.Errorf("reading rune: %w", err)
			}
		}

		value := literal.String()
		tokenType := token.Number
		if value == "%" {
			tokenType = token.Percent
		}
		return l.newTokenPosition(tokenType, value, pos), nil
	}
}

// processNumberCharacter processes a character as part of a number
// and detects common prefixes and suffixes of number bases.
// It returns whether the character was handled and the character was valid.
// nolint: cyclop
func (l *Lexer) processNumberCharacter(firstCharacter, r rune, i int, isBinary, hasPrefix *bool,
	literal *strings.Builder) (bool, bool) {

	switch {
	case *isBinary && r != '0' && r != '1':
		// in case the % operator is used and the following character
		// is not a 0 or 1, the meaning of % at this point will not be
		// the binary prefix but the % modulo operator
		return false, false

	case unicode.IsDigit(r):
		literal.WriteRune(r)
		return false, true

	case r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F':
		// handle immediate numbers like #$12 and #%10001000,
		// asm6 supports .hex c0 which has no prefix for the number.
		// the character b is disambiguous here as it can also be part of
		// a binary indicator suffix.
		literal.WriteRune(r)
		return false, true

	case i == 0 && r == '$': // hex prefix
		// do not convert it to 0x prefix as it can also be used as reference
		// to the current program counter
		literal.WriteRune(r)
		*hasPrefix = true
		return false, true

	case i == 1 && firstCharacter == '0' && (r == 'x' || r == 'X'): // hex prefix
		literal.WriteRune(r)
		*hasPrefix = true
		return false, true

	case i == 0 && r == '%': // binary prefix
		literal.WriteRune(r)
		*hasPrefix = true
		*isBinary = true
		return false, true

	case !*hasPrefix && (r == 'h' || r == 'H'): // hex suffix
		l.addHexPrefix(literal)
		return true, false

	default:
		return false, false
	}
}

// addHexPrefix adds a 0x hex number prefix, to allow Go to parse the number.
// This is used for numbers declared with a suffix like h.
func (l *Lexer) addHexPrefix(builder *strings.Builder) {
	literal := builder.String()
	builder.Reset()

	numberStart := 0
	if len(literal) > 0 {
		firstCharacter := rune(literal[0])
		if l.cfg.DecimalPrefix != 0 && firstCharacter == l.cfg.DecimalPrefix {
			builder.WriteRune(firstCharacter)
			numberStart = 1
		}
	}

	builder.WriteRune('0') // prepend 0x hex prefix
	builder.WriteRune('x')
	builder.WriteString(literal[numberStart:])
}

// readLiteral reads an identifier token.
func (l *Lexer) readLiteral(r rune) (token.Token, error) {
	var literal strings.Builder
	literal.WriteRune(r)
	pos := l.pos

	for {
		r, _, err := l.reader.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// return the current identifier, the next read will fail and be EOF
				return l.newTokenPosition(token.Identifier, literal.String(), pos), nil
			}

			return token.Token{}, fmt.Errorf("reading rune: %w", err)
		}

		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '$' || r == '_' || r == '-' || r == '"' || r == '\'' {
			l.pos.Column++
			literal.WriteRune(r)
			continue
		}

		if err := l.reader.UnreadRune(); err != nil {
			return token.Token{}, fmt.Errorf("unreading rune: %w", err)
		}
		return l.newTokenPosition(token.Identifier, literal.String(), pos), nil
	}
}

// readLiteral reads an escaped string token.
func (l *Lexer) readString(stringEscape rune) (token.Token, error) {
	var literal strings.Builder
	pos := l.pos
	previousRune := stringEscape
	literal.WriteRune(stringEscape)

	for {
		r, _, err := l.reader.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// return the current identifier, the next read will fail and be EOF
				return l.newTokenPosition(token.Identifier, literal.String(), pos), nil
			}

			return token.Token{}, fmt.Errorf("reading rune: %w", err)
		}

		l.pos.Column++
		literal.WriteRune(r)

		// write all characters until the string delimiter was found and is not escaped
		// by using \ for the previous character
		if r != stringEscape || previousRune == '\\' {
			continue
		}

		return l.newTokenPosition(token.Identifier, literal.String(), pos), nil
	}
}

// readComment reads until the end of the comment/line or file.
func (l *Lexer) readComment(prefix rune) (token.Token, error) {
	t := token.Token{
		Position: l.pos,
		Type:     token.Comment,
		Value:    "",
	}

	skipPrefixes := true

	for {
		r, _, err := l.reader.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				t.Value = strings.TrimSpace(t.Value)
				return t, nil
			}

			return token.Token{}, fmt.Errorf("reading rune: %w", err)
		}

		l.pos.Column++

		// skip multiple prefixes at the beginning of a comment for example when using //
		if skipPrefixes {
			if r == prefix {
				continue
			}

			skipPrefixes = false
		}

		switch r {
		case '\n':
			l.pos.NextLine()
			t.Value = strings.TrimSpace(t.Value)
			return t, nil

		default:
			t.Value += string(r)
		}
	}
}

// newToken returns a new token with the current position and an empty value.
func (l *Lexer) newToken(tokenType token.Type) token.Token {
	return token.Token{
		Position: l.pos,
		Type:     tokenType,
		Value:    "",
	}
}

// newToken returns a new token with the passed position and literal.
func (l *Lexer) newTokenPosition(tokenType token.Type, literal string, pos token.Position) token.Token {
	return token.Token{
		Position: pos,
		Type:     tokenType,
		Value:    literal,
	}
}
