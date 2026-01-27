// Package lexer provides lexical analysis for assembly language files.
//
// The lexer tokenizes assembly source code and configuration files, handling
// various number formats, comments, and language-specific constructs common
// in retro computer assembly languages.
//
// # Number Format Support
//
// The lexer supports multiple number formats commonly used in assembly:
//   - Decimal: 123, #123
//   - Hexadecimal: $FF, 0xFF, FFh
//   - Binary: %11110000
//
// # Configuration
//
// Lexer behavior is configurable through the Config struct:
//   - CommentPrefixes: Define comment delimiters (e.g., "//", ";")
//   - DecimalPrefix: Define immediate value prefix (e.g., "#")
//
// # Example Usage
//
//	cfg := lexer.Config{
//		CommentPrefixes: []string{"//", ";"},
//		DecimalPrefix:   '#',
//	}
//
//	reader := strings.NewReader("lda #$FF ; load accumulator")
//	lex := lexer.New(cfg, reader)
//
//	for {
//		token, err := lex.NextToken()
//		if err != nil {
//			return err
//		}
//		if token.Type == token.EOF {
//			break
//		}
//		// Process token
//	}
//
// # Error Handling
//
// The lexer uses structured errors with sentinel values for consistent
// error handling. All errors preserve the underlying cause while providing
// semantic context through error wrapping.
package lexer
