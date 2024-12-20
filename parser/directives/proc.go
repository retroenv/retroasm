package directives

import (
	"github.com/retroenv/retroasm/arch"
	"github.com/retroenv/retroasm/parser/ast"
)

// Proc ...
func Proc(p arch.Parser) (ast.Node, error) {
	p.AdvanceReadPosition(2)
	next := p.NextToken(0)
	if next.Type.IsTerminator() {
		return nil, errMissingParameter
	}

	return ast.NewFunction(next.Value), nil
}

// EndProc ...
func EndProc(p arch.Parser) (ast.Node, error) {
	p.AdvanceReadPosition(1)
	return ast.NewFunctionEnd(), nil
}
