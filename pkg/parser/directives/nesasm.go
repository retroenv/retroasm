package directives

import (
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/number"
	"github.com/retroenv/retroasm/pkg/parser/ast"
)

var nes2Directives = map[string]ast.ConfigurationItem{
	"nes2chrram":  ast.ConfigNes2ChrRAM,
	"nes2prgram":  ast.ConfigNes2PrgRAM,
	"nes2sub":     ast.ConfigNes2Sub,
	"nes2tv":      ast.ConfigNes2TV,
	"nes2vs":      ast.ConfigNes2VS,
	"nes2bram":    ast.ConfigNes2BRam,
	"nes2chrbram": ast.ConfigNes2ChrBRam,
}

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

// Nes2Config converts NES 2.0 header directives (asm6f) to ast configuration nodes.
func Nes2Config(p arch.Parser) (ast.Node, error) {
	next := p.NextToken(1)
	directive := strings.ToLower(next.Value)
	configItem, ok := nes2Directives[directive]
	if !ok {
		return nil, fmt.Errorf("unsupported NES 2.0 config item %s", next.Value)
	}

	value := p.NextToken(2)
	if value.Type != token.Number {
		return nil, fmt.Errorf("unsupported config value type %s", value.Type)
	}

	i, err := number.Parse(value.Value)
	if err != nil {
		return nil, fmt.Errorf("parsing number '%s': %w", value.Value, err)
	}

	p.AdvanceReadPosition(2)
	cfg := ast.NewConfiguration(configItem)
	cfg.Value = i
	return cfg, nil
}

// NesasmOffsetCounter parses a .rsset directive for setting the NESASM offset counter.
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
