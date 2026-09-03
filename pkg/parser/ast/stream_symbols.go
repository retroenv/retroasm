package ast

// RebuildSymbols atomically derives symbol metadata from stream entries.
func (stm *Stream) RebuildSymbols() error {
	if stm == nil {
		return ErrInvalidStream
	}

	currentSegment := ""
	symbols := make([]Symbol, 0)
	for index, entry := range stm.entries {
		if segment, ok := segmentFromNode(entry.Node); ok {
			currentSegment = segment.Name
			continue
		}

		symbol, ok := symbolFromEntry(index, currentSegment, entry)
		if ok {
			symbols = append(symbols, symbol)
		}
	}
	if err := validateSymbols(symbols, stm.entries); err != nil {
		return err
	}

	stm.symbols = symbols
	return nil
}

func symbolFromEntry(entryIndex int, segment string, entry Entry) (Symbol, bool) {
	symbol := Symbol{
		EntryIndex: entryIndex,
		Segment:    segment,
		Position:   entry.Position,
	}

	switch node := entry.Node.(type) {
	case Alias:
		return symbolFromAlias(symbol, node), true
	case *Alias:
		if node != nil {
			return symbolFromAlias(symbol, *node), true
		}
	case Label:
		symbol.Kind = LabelSymbol
		symbol.Name = node.Name
		symbol.Expression = NewLocationSymbolExpression()
		return symbol, true
	case *Label:
		if node != nil {
			symbol.Kind = LabelSymbol
			symbol.Name = node.Name
			symbol.Expression = NewLocationSymbolExpression()
			return symbol, true
		}
	case Function:
		symbol.Kind = FunctionSymbol
		symbol.Name = node.Name
		symbol.Expression = NewLocationSymbolExpression()
		return symbol, true
	case *Function:
		if node != nil {
			symbol.Kind = FunctionSymbol
			symbol.Name = node.Name
			symbol.Expression = NewLocationSymbolExpression()
			return symbol, true
		}
	}
	return Symbol{}, false
}

func symbolFromAlias(symbol Symbol, alias Alias) Symbol {
	symbol.Kind = EquSymbol
	if alias.SymbolReusable {
		symbol.Kind = AliasSymbol
	}
	symbol.Name = alias.Name
	symbol.Expression = NewDefinitionSymbolExpression(alias.Expression)
	return symbol
}

func segmentFromNode(node Node) (Segment, bool) {
	switch segment := node.(type) {
	case Segment:
		return segment, true
	case *Segment:
		if segment != nil {
			return *segment, true
		}
	}
	return Segment{}, false
}
