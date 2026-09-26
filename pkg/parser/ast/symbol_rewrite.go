package ast

import (
	"errors"
	"fmt"

	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
)

// InstructionArgumentSymbolRewriter rewrites symbols in an independent typed argument.
// Implementations must retain the instruction form and all non-symbol operand data.
type InstructionArgumentSymbolRewriter interface {
	RewriteInstructionSymbols(rename func(string) string) (any, error)
}

// RewriteNodeSymbols returns an independent node with renamed flat symbols.
// The callback receives each symbol definition and reference, without numeric tokens.
// Unexpanded scopes, macros, and opaque arguments without a rewrite contract are rejected.
func RewriteNodeSymbols(node Node, rename func(string) string) (Node, error) {
	if node == nil {
		return nil, nil //nolint:nilnil // An absent operand has no symbols.
	}
	if rename == nil {
		return nil, errors.New("symbol rewrite callback is nil")
	}
	return rewriteCopiedNodeSymbols(node.Copy(), rename)
}

func rewriteCopiedNodeSymbols(node Node, rename func(string) string) (Node, error) {
	switch value := node.(type) {
	case Label:
		value.Name = rename(value.Name)
		return value, nil
	case Identifier:
		value.Name = rename(value.Name)
		return value, nil
	case Function:
		value.Name = rename(value.Name)
		return value, nil
	case Variable:
		value.Name = rename(value.Name)
		return value, nil
	case Alias:
		value.Name = rename(value.Name)
		value.Expression = rewriteSymbolExpressionTokens(value.Expression, rename)
		return value, nil
	case Instruction:
		argument, err := RewriteNodeSymbols(value.Argument, rename)
		value.Argument = argument
		return value, err
	case InstructionArguments:
		for index, argument := range value.Values {
			rewritten, err := RewriteNodeSymbols(argument, rename)
			if err != nil {
				return nil, err
			}
			value.Values[index] = rewritten
		}
		return value, nil
	case InstructionArgument:
		return rewriteTypedArgumentSymbols(value, rename)
	case RegisterValue:
		rewritten, err := RewriteNodeSymbols(value.Value, rename)
		value.Value = rewritten
		return value, err
	case RegisterRegisterValue:
		rewritten, err := RewriteNodeSymbols(value.Value, rename)
		value.Value = rewritten
		return value, err
	default:
		return rewriteDirectiveSymbols(node, rename)
	}
}

func rewriteDirectiveSymbols(node Node, rename func(string) string) (Node, error) {
	switch value := node.(type) {
	case Expression:
		value.Value = rewriteSymbolExpressionTokens(value.Value, rename)
		return value, nil
	case Data:
		value.Size = rewriteSymbolExpressionTokens(value.Size, rename)
		for index, item := range value.Values {
			value.Values[index] = rewriteSymbolExpressionTokens(item, rename)
		}
		return value, nil
	case Base:
		value.Address = rewriteSymbolExpressionTokens(value.Address, rename)
		return value, nil
	case Enum:
		value.Address = rewriteSymbolExpressionTokens(value.Address, rename)
		return value, nil
	case Configuration:
		value.Expression = rewriteSymbolExpressionTokens(value.Expression, rename)
		return value, nil
	case If:
		value.Condition = rewriteSymbolExpressionTokens(value.Condition, rename)
		return value, nil
	case ElseIf:
		value.Condition = rewriteSymbolExpressionTokens(value.Condition, rename)
		return value, nil
	case Ifdef:
		value.Identifier = rename(value.Identifier)
		return value, nil
	case Ifndef:
		value.Identifier = rename(value.Identifier)
		return value, nil
	case Rept:
		value.Count = rewriteSymbolExpressionTokens(value.Count, rename)
		return value, nil
	case Number, Operator, *Comment, Segment, Bank, OffsetCounter, FunctionEnd, EnumEnd, Else, Endif, Endr:
		return node, nil
	default:
		return nil, fmt.Errorf("node %T has no flat symbol rewrite contract", node)
	}
}

func rewriteSymbolExpressionTokens(value *expression.Expression, rename func(string) string) *expression.Expression {
	if value == nil {
		return nil
	}
	copied := value.Copy()
	for index, item := range copied.Tokens() {
		if item.Type == token.Identifier {
			copied.Tokens()[index].Value = rename(item.Value)
		}
	}
	return copied
}

func rewriteTypedArgumentSymbols(value InstructionArgument, rename func(string) string) (Node, error) {
	rewriter, ok := value.Value.(InstructionArgumentSymbolRewriter)
	if !ok {
		return nil, fmt.Errorf("typed argument %T has no symbol rewrite contract", value.Value)
	}
	rewritten, err := rewriter.RewriteInstructionSymbols(rename)
	if err != nil {
		return nil, fmt.Errorf("rewriting typed argument: %w", err)
	}
	value.Value = rewritten
	return value, nil
}
