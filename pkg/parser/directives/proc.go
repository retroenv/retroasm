package directives

import (
	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

// Proc parses a .proc directive for defining a named procedure.
func Proc(p arch.Parser) (ast.Node, error) {
	p.AdvanceReadPosition(2)
	next := p.NextToken(0)
	if next.Type.IsTerminator() {
		return nil, errMissingParameter
	}

	return ast.NewFunction(next.Value), nil
}

// EndProc parses a .endproc directive for ending a procedure definition.
func EndProc(p arch.Parser) (ast.Node, error) {
	p.AdvanceReadPosition(1)
	return ast.NewFunctionEnd(), nil
}
