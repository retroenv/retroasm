package directives

import (
	"github.com/retroenv/assembler/lexer/token"
	"github.com/retroenv/assembler/parser/ast"
)

// ParseModifier ...
func ParseModifier(p Parser) []ast.Modifier {
	var modifiers []ast.Modifier
	var operator ast.Operator

	for next1 := p.NextToken(1); ; next1 = p.NextToken(1) {
		switch next1.Type {

		case token.Plus:
			operator = ast.Operator{
				Operator: "+",
			}
		case token.Minus:
			operator = ast.Operator{
				Operator: "-",
			}

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
