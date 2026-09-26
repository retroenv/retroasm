package parser

import (
	"fmt"

	"github.com/retroenv/retroasm/pkg/parser/ast"
)

// RewriteInstructionSymbols returns an independent argument with renamed symbols.
func (resolved ResolvedInstruction) RewriteInstructionSymbols(rename func(string) string) (any, error) {
	resolved = resolved.CopyInstructionArgument().(ResolvedInstruction)
	for _, address := range []*EffectiveAddress{resolved.SrcEA, resolved.DstEA} {
		if address == nil {
			continue
		}
		value, err := ast.RewriteNodeSymbols(address.Value, rename)
		if err != nil {
			return nil, fmt.Errorf("rewriting operand symbols: %w", err)
		}
		address.Value = value
	}
	return resolved, nil
}
