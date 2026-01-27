package directives

import (
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/number"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

var nesasmDirectives = map[string]ast.ConfigurationItem{
	"inesbat":    ast.ConfigBattery,
	"ineschr":    ast.ConfigChr,
	"inesmap":    ast.ConfigMapper,
	"inesmir":    ast.ConfigMirror,
	"inesprg":    ast.ConfigPrg,
	"inessubmap": ast.ConfigSubMapper,
}

// NesasmConfig converts nesasm control directives to ast configuration nodes.
func NesasmConfig(p arch.Parser) (ast.Node, error) {
	next := p.NextToken(1)
	directive := strings.ToLower(next.Value)
	configItem, ok := nesasmDirectives[directive]
	if !ok {
		return nil, fmt.Errorf("unsupported nesasm config item %s", next.Value)
	}

	value := p.NextToken(2)
	if value.Type != token.Number {
		return nil, fmt.Errorf("unsupported config value type %s", next.Type)
	}

	i, err := number.Parse(value.Value)
	if err != nil {
		return nil, fmt.Errorf("parsing number '%s': %w", value.Value, err)
	}

	switch configItem {
	case ast.ConfigChr:
		if i < 0xeff {
			i *= 8192
		}
	case ast.ConfigPrg:
		if i < 0xeff {
			i *= 16384
		}
	}

	p.AdvanceReadPosition(2)
	cfg := ast.NewConfiguration(configItem)
	cfg.Value = i
	return cfg, nil
}

// NesasmOffsetCounter ...
func NesasmOffsetCounter(p arch.Parser) (ast.Node, error) {
	value := p.NextToken(2)
	if value.Type != token.Number {
		return nil, fmt.Errorf("unsupported offset counter value type %s", value.Type)
	}

	i, err := number.Parse(value.Value)
	if err != nil {
		return nil, fmt.Errorf("parsing number '%s': %w", value.Value, err)
	}

	p.AdvanceReadPosition(2)
	return ast.NewOffsetCounter(i), nil
}
