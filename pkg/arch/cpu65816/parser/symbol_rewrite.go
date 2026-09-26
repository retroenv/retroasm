package parser

import (
	"fmt"

	"github.com/retroenv/retroasm/pkg/parser/ast"
)

// RewriteInstructionSymbols returns an independent argument with renamed symbols.
func (resolved ResolvedInstruction) RewriteInstructionSymbols(rename func(string) string) (any, error) {
	resolved = resolved.CopyInstructionArgument().(ResolvedInstruction)
	for index := range resolved.Operands {
		value, err := ast.RewriteNodeSymbols(resolved.Operands[index].Value, rename)
		if err != nil {
			return nil, fmt.Errorf("rewriting operand symbols: %w", err)
		}
		resolved.Operands[index].Value = value
	}
	return resolved, nil
}
