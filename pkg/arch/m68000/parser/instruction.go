package parser

import (
	"errors"
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/m68000"
)

// ParseIdentifier parses an M68000 instruction and returns the corresponding AST node.
func ParseIdentifier(p arch.Parser, ins *m68000.Instruction, mnemonic string) (ast.Node, error) { //nolint:cyclop,funlen // instruction dispatch requires many cases
	size := m68000.SizeWord // default size

	// Parse size suffix from the original mnemonic (e.g., "move.l" passed from Instruction lookup)
	_, sizeSuffix := ParseSizeSuffix(mnemonic)
	if sizeSuffix != 0 {
		size = sizeSuffix
	}

	// Parse condition code for Bcc/DBcc/Scc
	var condCode uint16
	_, cond, hasCond := ParseConditionCode(mnemonic)
	if hasCond {
		condCode = cond
	}

	// Check for size suffix in token stream: . B/W/L
	if p.NextToken(1).Type == token.Dot {
		if s := parseSizeToken(p.NextToken(2)); s != 0 {
			size = s
			p.AdvanceReadPosition(2) // skip . and size
		}
	}

	resolved := &ResolvedInstruction{
		Instruction: ins,
		Size:        size,
		Extra:       condCode,
	}

	// Check for no-operand instructions
	if isNoOperandInstruction(ins.Name) {
		return buildInstructionNode(ins.Name, resolved), nil
	}

	next := p.NextToken(1)
	if next.Type.IsTerminator() {
		return buildInstructionNode(ins.Name, resolved), nil
	}

	p.AdvanceReadPosition(1)

	// Parse based on instruction type
	var err error
	switch {
	case isBranchInstruction(ins.Name):
		err = parseBranch(p, resolved)
	case ins.Name == m68000.DBccName:
		err = parseDBcc(p, resolved)
	case ins.Name == m68000.MOVEMName:
		err = parseMOVEM(p, resolved)
	case ins.Name == m68000.MOVEQName:
		err = parseMOVEQ(p, resolved)
	case ins.Name == m68000.TRAPName:
		err = parseTRAP(p, resolved)
	case ins.Name == m68000.STOPName:
		err = parseSTOP(p, resolved)
	case ins.Name == m68000.LINKName:
		err = parseLINK(p, resolved)
	case ins.Name == m68000.UNLKName:
		err = parseUNLK(p, resolved)
	case ins.Name == m68000.SWAPName || ins.Name == m68000.EXTName:
		err = parseDataRegOnly(p, resolved)
	case ins.Name == m68000.EXGName:
		err = parseEXG(p, resolved)
	case isQuickInstruction(ins.Name):
		err = parseQuick(p, resolved)
	default:
		err = parseGenericOperands(p, resolved)
	}

	if err != nil {
		return nil, fmt.Errorf("parsing instruction '%s': %w", mnemonic, err)
	}

	return buildInstructionNode(ins.Name, resolved), nil
}

func buildInstructionNode(name string, resolved *ResolvedInstruction) ast.Node {
	argument := ast.NewInstructionArgument(*resolved)
	return ast.NewInstruction(name, int(m68000.NoAddressing), argument, nil)
}

func isNoOperandInstruction(name string) bool {
	switch name {
	case m68000.NOPName, m68000.RTSName, m68000.RTEName, m68000.RTRName,
		m68000.RESETName, m68000.TRAPVName, m68000.ILLEGALName:
		return true
	}
	return false
}

func isBranchInstruction(name string) bool {
	return name == m68000.BccName || name == m68000.BRAName || name == m68000.BSRName
}

func isQuickInstruction(name string) bool {
	return name == m68000.ADDQName || name == m68000.SUBQName
}

func parseBranch(p arch.Parser, resolved *ResolvedInstruction) error {
	ea, err := parseEffectiveAddress(p)
	if err != nil {
		return err
	}
	resolved.DstEA = ea
	return nil
}

func parseDBcc(p arch.Parser, resolved *ResolvedInstruction) error {
	// DBcc Dn,label
	ea, err := parseEffectiveAddress(p)
	if err != nil {
		return err
	}
	resolved.SrcEA = ea // data register

	if p.NextToken(1).Type != token.Comma {
		return errors.New("expected comma in DBcc")
	}
	p.AdvanceReadPosition(2) // skip ',' and advance

	dst, err := parseEffectiveAddress(p)
	if err != nil {
		return err
	}
	resolved.DstEA = dst // displacement/label
	return nil
}

func parseMOVEM(p arch.Parser, resolved *ResolvedInstruction) error {
	// MOVEM can be: MOVEM reglist,<ea> or MOVEM <ea>,reglist
	tok := p.NextToken(0)

	// Try to parse as register list first
	if isRegisterListStart(tok.Value) {
		regList, err := parseRegisterListFromTokens(p)
		if err != nil {
			return err
		}
		resolved.SrcEA = &EffectiveAddress{RegList: regList}

		if p.NextToken(1).Type != token.Comma {
			return errors.New("expected comma in MOVEM")
		}
		p.AdvanceReadPosition(2)

		dst, err := parseEffectiveAddress(p)
		if err != nil {
			return err
		}
		resolved.DstEA = dst
		resolved.Extra = 0 // register-to-memory
		return nil
	}

	// Parse as <ea>,reglist
	src, err := parseEffectiveAddress(p)
	if err != nil {
		return err
	}
	resolved.SrcEA = src

	if p.NextToken(1).Type != token.Comma {
		return errors.New("expected comma in MOVEM")
	}
	p.AdvanceReadPosition(2)

	regList, err := parseRegisterListFromTokens(p)
	if err != nil {
		return err
	}
	resolved.DstEA = &EffectiveAddress{RegList: regList}
	resolved.Extra = 1 // memory-to-register
	return nil
}

func isRegisterListStart(s string) bool {
	upper := strings.ToUpper(s)
	if len(upper) < 2 {
		return false
	}
	return (upper[0] == 'D' || upper[0] == 'A') && upper[1] >= '0' && upper[1] <= '7'
}

func parseRegisterListFromTokens(p arch.Parser) (uint16, error) {
	// Collect tokens that form the register list
	var parts []string
	tok := p.NextToken(0)
	parts = append(parts, tok.Value)

	for {
		next := p.NextToken(1)
		if next.Type == token.Comma || next.Type.IsTerminator() {
			break
		}
		// Continue collecting: could be '-', '/', register names
		if next.Value == "/" || next.Value == "-" || isRegisterListStart(next.Value) {
			p.AdvanceReadPosition(1)
			parts = append(parts, next.Value)
			continue
		}
		break
	}

	return parseRegisterList(strings.Join(parts, ""))
}

func parseMOVEQ(p arch.Parser, resolved *ResolvedInstruction) error {
	// MOVEQ #imm,Dn
	src, err := parseEffectiveAddress(p)
	if err != nil {
		return err
	}
	resolved.SrcEA = src

	if p.NextToken(1).Type != token.Comma {
		return errors.New("expected comma in MOVEQ")
	}
	p.AdvanceReadPosition(2)

	dst, err := parseEffectiveAddress(p)
	if err != nil {
		return err
	}
	resolved.DstEA = dst
	resolved.Size = m68000.SizeLong
	return nil
}

func parseTRAP(p arch.Parser, resolved *ResolvedInstruction) error {
	// TRAP #vector
	ea, err := parseEffectiveAddress(p)
	if err != nil {
		return err
	}
	resolved.SrcEA = ea
	return nil
}

func parseSTOP(p arch.Parser, resolved *ResolvedInstruction) error {
	// STOP #imm
	ea, err := parseEffectiveAddress(p)
	if err != nil {
		return err
	}
	resolved.SrcEA = ea
	return nil
}

func parseLINK(p arch.Parser, resolved *ResolvedInstruction) error {
	// LINK An,#displacement
	src, err := parseEffectiveAddress(p)
	if err != nil {
		return err
	}
	resolved.SrcEA = src

	if p.NextToken(1).Type != token.Comma {
		return errors.New("expected comma in LINK")
	}
	p.AdvanceReadPosition(2)

	dst, err := parseEffectiveAddress(p)
	if err != nil {
		return err
	}
	resolved.DstEA = dst
	return nil
}

func parseUNLK(p arch.Parser, resolved *ResolvedInstruction) error {
	// UNLK An
	ea, err := parseEffectiveAddress(p)
	if err != nil {
		return err
	}
	resolved.DstEA = ea
	return nil
}

func parseDataRegOnly(p arch.Parser, resolved *ResolvedInstruction) error {
	// SWAP Dn, EXT Dn
	ea, err := parseEffectiveAddress(p)
	if err != nil {
		return err
	}
	resolved.DstEA = ea
	return nil
}

func parseEXG(p arch.Parser, resolved *ResolvedInstruction) error {
	// EXG Rx,Ry
	src, err := parseEffectiveAddress(p)
	if err != nil {
		return err
	}
	resolved.SrcEA = src

	if p.NextToken(1).Type != token.Comma {
		return errors.New("expected comma in EXG")
	}
	p.AdvanceReadPosition(2)

	dst, err := parseEffectiveAddress(p)
	if err != nil {
		return err
	}
	resolved.DstEA = dst
	return nil
}

func parseQuick(p arch.Parser, resolved *ResolvedInstruction) error {
	// ADDQ #data,<ea> / SUBQ #data,<ea>
	src, err := parseEffectiveAddress(p)
	if err != nil {
		return err
	}
	resolved.SrcEA = src

	if p.NextToken(1).Type != token.Comma {
		return errors.New("expected comma in quick instruction")
	}
	p.AdvanceReadPosition(2)

	dst, err := parseEffectiveAddress(p)
	if err != nil {
		return err
	}
	resolved.DstEA = dst
	return nil
}

func parseGenericOperands(p arch.Parser, resolved *ResolvedInstruction) error {
	src, err := parseEffectiveAddress(p)
	if err != nil {
		return err
	}

	if p.NextToken(1).Type != token.Comma {
		// Single operand instruction
		resolved.DstEA = src
		return nil
	}

	// Two operand instruction
	resolved.SrcEA = src
	p.AdvanceReadPosition(2)

	dst, err := parseEffectiveAddress(p)
	if err != nil {
		return err
	}
	resolved.DstEA = dst
	return nil
}
