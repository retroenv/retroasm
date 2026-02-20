package directives

import (
	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

// ParseModifier parses instruction address modifiers (+/- offset expressions).
func ParseModifier(p arch.Parser) []ast.Modifier {
	var modifiers []ast.Modifier
	var operator ast.Operator

	for next1 := p.NextToken(1); ; next1 = p.NextToken(1) {
		switch next1.Type {
		case token.Plus:
			operator = ast.NewOperator("+")
		case token.Minus:
			operator = ast.NewOperator("-")
		default:
			return modifiers
		}

		param := p.NextToken(2)
		modifier := ast.Modifier{
			Operator: operator,
			Value:    param.Value,
		}
		modifiers = append(modifiers, modifier)
		p.AdvanceReadPosition(2)
	}
}
