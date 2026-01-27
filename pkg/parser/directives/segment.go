package directives

import (
	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

// Segment ...
func Segment(p arch.Parser) (ast.Node, error) {
	next := p.NextToken(2)
	if next.Type.IsTerminator() {
		return nil, errMissingParameter
	}
	p.AdvanceReadPosition(2)

	return ast.NewSegment(next.Value), nil
}
