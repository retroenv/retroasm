package codec

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

func (c *Codec[T]) formatStructuralNode(node ast.Node) (string, error) {
	switch typed := node.(type) {
	case ast.Scope:
		return c.formatScope(typed)
	case ast.ScopeEnd:
		if c.configuration.CompatibilityMode != config.CompatCa65 {
			return "", fmt.Errorf("%w: scope end in %s mode", ErrFormattingUnsupported, c.configuration.CompatibilityMode)
		}
		return ".endscope", nil
	case ast.Function:
		if !validIdentifier(typed.Name) {
			return "", fmt.Errorf("%w: function name %q", ErrFormattingUnsupported, typed.Name)
		}
		return ".proc " + typed.Name, nil
	case ast.FunctionEnd:
		return ".endproc", nil
	case ast.Enum:
		return formatExpressionDirective(".enum", typed.Address)
	case ast.EnumEnd:
		return ".ende", nil
	case ast.Rept:
		return formatExpressionDirective(".rept", typed.Count)
	case ast.Endr:
		return ".endr", nil
	default:
		return c.formatConditionalNode(node)
	}
}

func (c *Codec[T]) formatConditionalNode(node ast.Node) (string, error) {
	switch typed := node.(type) {
	case ast.If:
		return formatExpressionDirective(".if", typed.Condition)
	case ast.Ifdef:
		if !validIdentifier(typed.Identifier) {
			return "", fmt.Errorf("%w: ifdef identifier %q", ErrFormattingUnsupported, typed.Identifier)
		}
		return ".ifdef " + typed.Identifier, nil
	case ast.Ifndef:
		if !validIdentifier(typed.Identifier) {
			return "", fmt.Errorf("%w: ifndef identifier %q", ErrFormattingUnsupported, typed.Identifier)
		}
		return ".ifndef " + typed.Identifier, nil
	case ast.Else:
		return ".else", nil
	case ast.ElseIf:
		return formatExpressionDirective(".elseif", typed.Condition)
	case ast.Endif:
		return ".endif", nil
	default:
		return c.formatSourceNode(node)
	}
}

func (c *Codec[T]) formatSourceNode(node ast.Node) (string, error) {
	switch typed := node.(type) {
	case ast.Include:
		return formatInclude(typed)
	case ast.Macro:
		return c.formatMacro(typed)
	case ast.Error:
		return formatError(typed)
	default:
		return "", fmt.Errorf("%w: %T", ErrFormattingUnsupported, node)
	}
}

func (c *Codec[T]) formatScope(scope ast.Scope) (string, error) {
	if c.configuration.CompatibilityMode != config.CompatCa65 {
		return "", fmt.Errorf("%w: scope in %s mode", ErrFormattingUnsupported, c.configuration.CompatibilityMode)
	}
	if scope.Name == "" {
		return ".scope", nil
	}
	if !validIdentifier(scope.Name) {
		return "", fmt.Errorf("%w: scope name %q", ErrFormattingUnsupported, scope.Name)
	}
	return ".scope " + scope.Name, nil
}

func (c *Codec[T]) formatMacro(macro ast.Macro) (string, error) {
	if !validIdentifier(macro.Name) {
		return "", fmt.Errorf("%w: macro name %q", ErrFormattingUnsupported, macro.Name)
	}
	for _, argument := range macro.Arguments {
		if !validIdentifier(argument) {
			return "", fmt.Errorf("%w: macro argument %q", ErrFormattingUnsupported, argument)
		}
	}
	if len(macro.Token) == 0 ||
		macro.Token[0].Type != token.EOL ||
		macro.Token[len(macro.Token)-1].Type != token.EOL {

		return "", fmt.Errorf("%w: macro body must start and end with a line boundary", ErrFormattingUnsupported)
	}

	header := ".macro " + macro.Name
	if c.configuration.CompatibilityMode == config.CompatNesasm {
		if len(macro.Arguments) != 0 {
			return "", fmt.Errorf("%w: NESASM macro has named arguments", ErrFormattingUnsupported)
		}
		header = macro.Name + " .macro"
	} else if len(macro.Arguments) != 0 {
		header += " " + strings.Join(macro.Arguments, ", ")
	}

	body, err := formatMacroTokens(macro.Token)
	if err != nil {
		return "", err
	}
	return header + body + ".endm", nil
}

func formatExpressionDirective(directive string, value *expression.Expression) (string, error) {
	formatted, err := ast.FormatExpression(value)
	if err != nil {
		return "", fmt.Errorf("%w: formatting %s expression: %w", ErrFormattingUnsupported, directive, err)
	}
	return directive + " " + formatted, nil
}

func formatInclude(include ast.Include) (string, error) {
	if !validIncludeName(include.Name) {
		return "", fmt.Errorf("%w: include name %q", ErrFormattingUnsupported, include.Name)
	}
	if include.Start < 0 || include.Size < 0 {
		return "", fmt.Errorf(
			"%w: negative include range %d,%d",
			ErrFormattingUnsupported,
			include.Start,
			include.Size,
		)
	}

	directive := ".include"
	if include.Binary {
		directive = ".incbin"
	}
	line := directive + " " + include.Name
	if include.Size != 0 {
		return line + ", " + strconv.Itoa(include.Start) + ", " + strconv.Itoa(include.Size), nil
	}
	if include.Start != 0 {
		return line + ", " + strconv.Itoa(include.Start), nil
	}
	return line, nil
}

func formatError(errorNode ast.Error) (string, error) {
	if strings.ContainsAny(errorNode.Message, "\"\\\r\n") {
		return "", fmt.Errorf("%w: error message requires escaping", ErrFormattingUnsupported)
	}
	return ".error " + strconv.Quote(errorNode.Message), nil
}

func formatMacroTokens(tokens []token.Token) (string, error) {
	var builder strings.Builder
	lineStart := true

	for _, bodyToken := range tokens {
		switch bodyToken.Type {
		case token.EOL:
			builder.WriteByte('\n')
			lineStart = true
		case token.Comment:
			if !lineStart {
				builder.WriteByte(' ')
			}
			builder.WriteByte(';')
			if bodyToken.Value != "" {
				builder.WriteByte(' ')
				builder.WriteString(bodyToken.Value)
			}
			// The lexer consumes the source newline as part of a comment token.
			builder.WriteByte('\n')
			lineStart = true
		case token.EOF, token.Illegal:
			return "", fmt.Errorf("%w: macro body token %s", ErrFormattingUnsupported, bodyToken.Type)
		default:
			value, err := formatMacroToken(bodyToken)
			if err != nil {
				return "", err
			}
			if !lineStart {
				builder.WriteByte(' ')
			}
			builder.WriteString(value)
			lineStart = false
		}
	}
	return builder.String(), nil
}

func formatMacroToken(bodyToken token.Token) (string, error) {
	if bodyToken.Type == token.Number || bodyToken.Type == token.Identifier {
		if bodyToken.Value == "" {
			return "", fmt.Errorf("%w: empty macro token %s", ErrFormattingUnsupported, bodyToken.Type)
		}
		return bodyToken.Value, nil
	}
	if bodyToken.Type < token.Dot || bodyToken.Type > token.BitwiseXor {
		return "", fmt.Errorf("%w: macro body token %s", ErrFormattingUnsupported, bodyToken.Type)
	}
	return bodyToken.Type.String(), nil
}

func validIncludeName(value string) bool {
	if validQuotedValue(value) {
		return true
	}
	base, extension, found := strings.Cut(value, ".")
	return found && !strings.Contains(extension, ".") && validIdentifier(base) && validIdentifier(extension)
}
