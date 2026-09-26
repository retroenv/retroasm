package ast

import "fmt"

// RenameSymbols atomically renames flat symbol definitions and references.
// Renames are simultaneous. A rename must not merge different symbol names.
// Entry positions, annotations, boundaries, comments, indices, and state stay intact.
func (stm *Stream) RenameSymbols(names map[string]string) error {
	if err := stm.Validate(); err != nil {
		return err
	}
	for source, target := range names {
		if source == "" || target == "" {
			return fmt.Errorf("%w: a symbol rename has an empty name", ErrInvalidStream)
		}
	}
	if len(names) == 0 {
		return nil
	}

	candidate := stm.Copy()
	seen := make(map[string]string)
	var collision error
	rename := func(source string) string {
		target, ok := names[source]
		if !ok {
			target = source
		}
		if previous, exists := seen[target]; exists && previous != source {
			collision = fmt.Errorf("%w: symbols %q and %q map to %q", ErrInvalidStream, previous, source, target)
		}
		seen[target] = source
		return target
	}
	for index := range candidate.entries {
		node, err := RewriteNodeSymbols(candidate.entries[index].Node, rename)
		if err != nil {
			return fmt.Errorf("%w: renaming entry %d: %w", ErrInvalidStream, index, err)
		}
		candidate.entries[index].Node = node
	}
	for index := range candidate.symbols {
		symbol := &candidate.symbols[index]
		symbol.Name = rename(symbol.Name)
		rewriteSymbolExpression(&symbol.Expression, rename)
	}
	for index := range candidate.relocations {
		rewriteSymbolExpression(&candidate.relocations[index].Expression, rename)
	}
	if collision != nil {
		return collision
	}
	if err := candidate.Validate(); err != nil {
		return fmt.Errorf("validating renamed stream: %w", err)
	}
	candidate.revision = &streamRevision{}
	*stm = *candidate
	return nil
}

func rewriteSymbolExpression(value *SymbolExpression, rename func(string) string) {
	if value.Symbol != "" {
		value.Symbol = rename(value.Symbol)
	}
	value.Definition = rewriteSymbolExpressionTokens(value.Definition, rename)
}
