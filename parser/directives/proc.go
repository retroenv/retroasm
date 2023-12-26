package directives

import "github.com/retroenv/assembler/parser/ast"

// Proc ...
func Proc(p Parser) (ast.Node, error) {
	p.AdvanceReadPosition(2)
	next := p.NextToken(0)
	if next.Type.IsTerminator() {
		return nil, errMissingParameter
	}

	return ast.NewFunction(next.Value), nil
}

// EndProc ...
func EndProc(p Parser) (ast.Node, error) {
	p.AdvanceReadPosition(1)
	return ast.NewFunctionEnd(), nil
}
