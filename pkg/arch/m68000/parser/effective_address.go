package parser

import (
	"errors"
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/number"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/m68000"
)

var (
	errMissingOperand     = errors.New("missing operand")
	errMissingCloseParen  = errors.New("missing closing parenthesis")
	errInvalidEA          = errors.New("invalid effective address")
)

// parseEffectiveAddress parses an effective address operand from the token stream.
func parseEffectiveAddress(p arch.Parser) (*EffectiveAddress, error) {
	tok := p.NextToken(0)

	switch {
	case tok.Type == token.Number && len(tok.Value) > 1 && tok.Value[0] == '#':
		return parseImmediateEA(tok)

	case tok.Value == "#":
		p.AdvanceReadPosition(1)
		return parseImmediateHashEA(p)

	case tok.Value == "-" && p.NextToken(1).Type == token.LeftParentheses:
		return parsePreDecrementEA(p)

	case tok.Type == token.LeftParentheses:
		return parseIndirectEA(p)

	case tok.Type == token.Number:
		return parseDisplacementOrAbsoluteEA(p, tok)

	case tok.Type == token.Identifier:
		return parseIdentifierEA(p, tok)

	case tok.Type.IsTerminator():
		return nil, errMissingOperand

	default:
		return nil, fmt.Errorf("%w: unexpected token %s", errInvalidEA, tok.Type)
	}
}

func parseImmediateEA(tok token.Token) (*EffectiveAddress, error) {
	valueStr := tok.Value[1:] // strip #
	v, err := number.Parse(valueStr)
	if err != nil {
		return nil, fmt.Errorf("parsing immediate value '%s': %w", valueStr, err)
	}
	return &EffectiveAddress{
		Mode:  m68000.ImmediateMode,
		Value: ast.NewNumber(v),
	}, nil
}

func parseImmediateHashEA(p arch.Parser) (*EffectiveAddress, error) {
	tok := p.NextToken(0)
	switch tok.Type {
	case token.Number:
		v, err := number.Parse(tok.Value)
		if err != nil {
			return nil, fmt.Errorf("parsing immediate value '%s': %w", tok.Value, err)
		}
		return &EffectiveAddress{
			Mode:  m68000.ImmediateMode,
			Value: ast.NewNumber(v),
		}, nil
	case token.Identifier:
		return &EffectiveAddress{
			Mode:  m68000.ImmediateMode,
			Value: ast.NewLabel(tok.Value),
		}, nil
	default:
		return nil, fmt.Errorf("expected immediate value after #, got %s", tok.Type)
	}
}

func parsePreDecrementEA(p arch.Parser) (*EffectiveAddress, error) {
	// Current is '-', next is '('
	p.AdvanceReadPosition(2) // skip '-' and '('
	regTok := p.NextToken(0)
	info, ok := lookupRegister(regTok.Value)
	if !ok || !info.isAddr || info.special {
		return nil, fmt.Errorf("expected address register in pre-decrement, got '%s'", regTok.Value)
	}
	p.AdvanceReadPosition(1) // skip register
	if p.NextToken(0).Type != token.RightParentheses {
		return nil, errMissingCloseParen
	}
	return &EffectiveAddress{
		Mode:     m68000.PreDecrementMode,
		Register: info.number,
	}, nil
}

func parseIndirectEA(p arch.Parser) (*EffectiveAddress, error) {
	// Current token is '('
	inner := p.NextToken(1)
	if inner.Type.IsTerminator() {
		return nil, errMissingOperand
	}

	if inner.Type == token.Identifier {
		return parseIndirectIdentifierEA(p, inner)
	}
	if inner.Type == token.Number {
		return parseIndirectNumberEA(p, inner)
	}

	return nil, fmt.Errorf("%w: unexpected token in indirect '%s'", errInvalidEA, inner.Type)
}

func parseIndirectIdentifierEA(p arch.Parser, regTok token.Token) (*EffectiveAddress, error) {
	info, ok := lookupRegister(regTok.Value)
	if !ok {
		// Could be a label in parentheses: (label) for absolute addressing
		return parseParenAbsoluteEA(p, regTok)
	}

	if !info.isAddr && !info.special {
		return nil, fmt.Errorf("expected address register in indirect, got '%s'", regTok.Value)
	}

	next := p.NextToken(2)
	switch next.Type {
	case token.RightParentheses:
		p.AdvanceReadPosition(2) // skip '(' register ')'
		if info.number == regPC {
			return &EffectiveAddress{Mode: m68000.PCDisplacementMode, Value: ast.NewNumber(0)}, nil
		}
		// Check for post-increment: (An)+
		if p.NextToken(1).Value == "+" {
			p.AdvanceReadPosition(1) // skip '+'
			return &EffectiveAddress{
				Mode:     m68000.PostIncrementMode,
				Register: info.number,
			}, nil
		}
		return &EffectiveAddress{
			Mode:     m68000.AddrRegIndirectMode,
			Register: info.number,
		}, nil

	case token.Comma:
		// Indexed mode: (An,Xn) or (PC,Xn)
		p.AdvanceReadPosition(3) // skip '(' register ','
		return parseIndexedEA(p, info)

	default:
		return nil, fmt.Errorf("%w: unexpected token after register in indirect '%s'", errInvalidEA, next.Type)
	}
}

func parseParenAbsoluteEA(p arch.Parser, labelTok token.Token) (*EffectiveAddress, error) {
	// (label) -> absolute addressing
	next := p.NextToken(2)
	if next.Type != token.RightParentheses {
		return nil, fmt.Errorf("%w: expected ')' after label in indirect", errInvalidEA)
	}
	p.AdvanceReadPosition(2) // skip '(' label ')'

	// Check for .W or .L suffix
	if p.NextToken(1).Type == token.Dot {
		suffix := p.NextToken(2)
		upper := strings.ToUpper(suffix.Value)
		if upper == "W" {
			p.AdvanceReadPosition(2) // skip '.W'
			return &EffectiveAddress{
				Mode:  m68000.AbsShortMode,
				Value: ast.NewLabel(labelTok.Value),
			}, nil
		}
		if upper == "L" {
			p.AdvanceReadPosition(2) // skip '.L'
			return &EffectiveAddress{
				Mode:  m68000.AbsLongMode,
				Value: ast.NewLabel(labelTok.Value),
			}, nil
		}
	}

	return &EffectiveAddress{
		Mode:  m68000.AbsLongMode,
		Value: ast.NewLabel(labelTok.Value),
	}, nil
}

func parseIndirectNumberEA(p arch.Parser, numTok token.Token) (*EffectiveAddress, error) {
	// (value) - absolute addressing
	next := p.NextToken(2)
	if next.Type != token.RightParentheses {
		return nil, fmt.Errorf("%w: expected ')' after number in indirect", errInvalidEA)
	}

	v, err := number.Parse(numTok.Value)
	if err != nil {
		return nil, fmt.Errorf("parsing address '%s': %w", numTok.Value, err)
	}

	p.AdvanceReadPosition(2) // skip '(' number ')'

	// Check for .W or .L suffix
	if p.NextToken(1).Type == token.Dot {
		suffix := p.NextToken(2)
		upper := strings.ToUpper(suffix.Value)
		if upper == "W" {
			p.AdvanceReadPosition(2)
			return &EffectiveAddress{Mode: m68000.AbsShortMode, Value: ast.NewNumber(v)}, nil
		}
		if upper == "L" {
			p.AdvanceReadPosition(2)
			return &EffectiveAddress{Mode: m68000.AbsLongMode, Value: ast.NewNumber(v)}, nil
		}
	}

	if v <= 0xFFFF {
		return &EffectiveAddress{Mode: m68000.AbsShortMode, Value: ast.NewNumber(v)}, nil
	}
	return &EffectiveAddress{Mode: m68000.AbsLongMode, Value: ast.NewNumber(v)}, nil
}

func parseIndexedEA(p arch.Parser, base registerInfo) (*EffectiveAddress, error) {
	// We already consumed '(' base ','
	// Expect: Xn.s ')'
	indexTok := p.NextToken(0)
	indexInfo, ok := lookupRegister(indexTok.Value)
	if !ok {
		return nil, fmt.Errorf("expected index register, got '%s'", indexTok.Value)
	}

	indexSize := m68000.SizeWord // default index size
	p.AdvanceReadPosition(1)    // skip index register

	// Check for .W or .L size suffix on index register
	if p.NextToken(0).Type == token.Dot {
		suffix := p.NextToken(1)
		upper := strings.ToUpper(suffix.Value)
		if upper == "W" {
			indexSize = m68000.SizeWord
			p.AdvanceReadPosition(2) // skip '.W'
		} else if upper == "L" {
			indexSize = m68000.SizeLong
			p.AdvanceReadPosition(2) // skip '.L'
		}
	}

	if p.NextToken(0).Type != token.RightParentheses {
		return nil, errMissingCloseParen
	}

	mode := m68000.IndexedMode
	if base.number == regPC {
		mode = m68000.PCIndexedMode
	}

	return &EffectiveAddress{
		Mode:      mode,
		Register:  base.number,
		IndexReg:  indexInfo.number,
		IndexSize: indexSize,
		IsAddrReg: indexInfo.isAddr,
		Value:     ast.NewNumber(0),
	}, nil
}

func parseDisplacementOrAbsoluteEA(p arch.Parser, numTok token.Token) (*EffectiveAddress, error) {
	v, err := number.Parse(numTok.Value)
	if err != nil {
		return nil, fmt.Errorf("parsing number '%s': %w", numTok.Value, err)
	}

	next := p.NextToken(1)

	// Check for displacement: value(An) or value(PC) or value(An,Xn.s)
	if next.Type == token.LeftParentheses {
		p.AdvanceReadPosition(2) // skip number '('
		return parseDisplacementEA(p, v)
	}

	// Check for .W or .L suffix for absolute
	if next.Type == token.Dot {
		suffix := p.NextToken(2)
		upper := strings.ToUpper(suffix.Value)
		if upper == "W" {
			p.AdvanceReadPosition(2) // skip '.W'
			return &EffectiveAddress{Mode: m68000.AbsShortMode, Value: ast.NewNumber(v)}, nil
		}
		if upper == "L" {
			p.AdvanceReadPosition(2) // skip '.L'
			return &EffectiveAddress{Mode: m68000.AbsLongMode, Value: ast.NewNumber(v)}, nil
		}
	}

	// Default: absolute (short if fits, long otherwise)
	if v <= 0xFFFF {
		return &EffectiveAddress{Mode: m68000.AbsShortMode, Value: ast.NewNumber(v)}, nil
	}
	return &EffectiveAddress{Mode: m68000.AbsLongMode, Value: ast.NewNumber(v)}, nil
}

func parseDisplacementEA(p arch.Parser, disp uint64) (*EffectiveAddress, error) {
	// Current position after number '(' - next is register
	regTok := p.NextToken(0)
	info, ok := lookupRegister(regTok.Value)
	if !ok {
		return nil, fmt.Errorf("expected register in displacement, got '%s'", regTok.Value)
	}

	next := p.NextToken(1)
	switch next.Type {
	case token.RightParentheses:
		p.AdvanceReadPosition(1) // skip register ')'
		mode := m68000.DisplacementMode
		if info.number == regPC {
			mode = m68000.PCDisplacementMode
		}
		return &EffectiveAddress{
			Mode:     mode,
			Register: info.number,
			Value:    ast.NewNumber(disp),
		}, nil

	case token.Comma:
		// Indexed: d8(An,Xn.s)
		p.AdvanceReadPosition(2) // skip register ','
		ea, err := parseIndexedEA(p, info)
		if err != nil {
			return nil, err
		}
		ea.Value = ast.NewNumber(disp)
		return ea, nil

	default:
		return nil, fmt.Errorf("expected ')' or ',' after register in displacement, got '%s'", next.Type)
	}
}

func parseIdentifierEA(p arch.Parser, tok token.Token) (*EffectiveAddress, error) {
	info, ok := lookupRegister(tok.Value)
	if !ok {
		// Not a register - treat as label (absolute address)
		return parseLabelEA(p, tok)
	}

	if info.special {
		if info.number == regSR {
			return &EffectiveAddress{Mode: m68000.StatusRegMode, Register: regSR}, nil
		}
		if info.number == regCCR {
			return &EffectiveAddress{Mode: m68000.StatusRegMode, Register: regCCR}, nil
		}
		if info.number == regUSP {
			return &EffectiveAddress{Mode: m68000.AddrRegDirectMode, Register: regUSP}, nil
		}
		return nil, fmt.Errorf("unexpected special register '%s'", tok.Value)
	}

	if info.isAddr {
		return &EffectiveAddress{Mode: m68000.AddrRegDirectMode, Register: info.number}, nil
	}
	return &EffectiveAddress{Mode: m68000.DataRegDirectMode, Register: info.number}, nil
}

func parseLabelEA(p arch.Parser, tok token.Token) (*EffectiveAddress, error) {
	next := p.NextToken(1)

	// Check for displacement: label(An) or label(PC)
	if next.Type == token.LeftParentheses {
		p.AdvanceReadPosition(2) // skip label '('
		regTok := p.NextToken(0)
		info, ok := lookupRegister(regTok.Value)
		if ok && (info.isAddr || info.number == regPC) {
			next2 := p.NextToken(1)
			if next2.Type == token.RightParentheses {
				p.AdvanceReadPosition(1) // skip register ')'
				mode := m68000.DisplacementMode
				if info.number == regPC {
					mode = m68000.PCDisplacementMode
				}
				return &EffectiveAddress{
					Mode:     mode,
					Register: info.number,
					Value:    ast.NewLabel(tok.Value),
				}, nil
			}
			if next2.Type == token.Comma {
				// label(An,Xn.s) or label(PC,Xn.s)
				p.AdvanceReadPosition(2) // skip register ','
				ea, err := parseIndexedEA(p, info)
				if err != nil {
					return nil, err
				}
				ea.Value = ast.NewLabel(tok.Value)
				return ea, nil
			}
		}
		// Not a valid displacement, backtrack is complex - treat as label
		return nil, fmt.Errorf("%w: unexpected token after label '(' '%s'", errInvalidEA, regTok.Value)
	}

	// Check for .W or .L suffix
	if next.Type == token.Dot {
		suffix := p.NextToken(2)
		upper := strings.ToUpper(suffix.Value)
		if upper == "W" {
			p.AdvanceReadPosition(2)
			return &EffectiveAddress{Mode: m68000.AbsShortMode, Value: ast.NewLabel(tok.Value)}, nil
		}
		if upper == "L" {
			p.AdvanceReadPosition(2)
			return &EffectiveAddress{Mode: m68000.AbsLongMode, Value: ast.NewLabel(tok.Value)}, nil
		}
	}

	// Default: label as absolute long address
	return &EffectiveAddress{Mode: m68000.AbsLongMode, Value: ast.NewLabel(tok.Value)}, nil
}
