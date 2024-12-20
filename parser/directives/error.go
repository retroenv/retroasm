package directives

import (
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/arch"
	"github.com/retroenv/retroasm/lexer/token"
	"github.com/retroenv/retroasm/parser/ast"
)

// Error ...
func Error(p arch.Parser) (ast.Node, error) {
	msg := p.NextToken(2)
	if msg.Type != token.Identifier {
		return nil, fmt.Errorf("unsupported error message type %s", msg.Type)
	}

	p.AdvanceReadPosition(2)
	return ast.NewError(strings.Trim(msg.Value, "\"'")), nil
}
