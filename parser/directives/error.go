package directives

import (
	"fmt"
	"strings"

	"github.com/retroenv/assembler/lexer/token"
	"github.com/retroenv/assembler/parser/ast"
)

// Error ...
func Error(p Parser) (ast.Node, error) {
	msg := p.NextToken(2)
	if msg.Type != token.Identifier {
		return nil, fmt.Errorf("unsupported error message type %s", msg.Type)
	}

	p.AdvanceReadPosition(2)
	return &ast.Error{
		Message: strings.Trim(msg.Value, "\"'"),
	}, nil
}
