package parser

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/number"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retroasm/pkg/parser/directives"
	"github.com/retroenv/retrogolib/arch/cpu/m65816"
)

var errMissingParameter = errors.New("missing parameter")

// ParseIdentifier parses an instruction identifier and returns an AST node.
func ParseIdentifier(parser arch.Parser, ins *m65816.Instruction) (ast.Node, error) {
	if len(ins.Addressing) == 1 && ins.HasAddressing(m65816.ImpliedAddressing) {
		return ast.NewInstruction(ins.Name, int(m65816.ImpliedAddressing), nil, nil), nil
	}

	node, err := parseInstruction(parser, ins)
	if err != nil {
		return nil, fmt.Errorf("parsing instruction %s: %w", ins.Name, err)
	}
	return node, nil
}

type instruction struct {
	instruction    *m65816.Instruction
	addressingSize addressingSize
	modifiers      []ast.Modifier
	arg1           token.Token
	arg2           token.Token
}

func parseInstruction(parser arch.Parser, instructionDetails *m65816.Instruction) (ast.Node, error) {
	parser.AdvanceReadPosition(1)

	var err error
	ins := &instruction{
		instruction: instructionDetails,
	}

	ins.addressingSize, err = parseAddressSize(parser, instructionDetails)
	if err != nil {
		return nil, fmt.Errorf("parsing addressing size: %w", err)
	}

	ins.arg1 = resolveArg1Token(parser)
	ins.modifiers = directives.ParseModifier(parser)

	next1 := parser.NextToken(1)
	if next1.Type == token.Comma {
		// Block move instructions take two number operands, not register suffixes
		if ins.instruction.HasAddressing(m65816.BlockMoveAddressing) {
			return parseBlockMove(parser, ins)
		}

		parser.AdvanceReadPosition(2)
		ins.arg2 = parser.NextToken(0)
		return parseInstructionSecondIdentifier(ins, false)
	}

	switch {
	case ins.arg1.Type == token.LeftParentheses:
		ins.arg1 = next1
		return parseInstructionParentheses(parser, ins)

	case ins.arg1.Type == token.LeftBracket:
		ins.arg1 = next1
		return parseInstructionBrackets(parser, ins)

	case ins.arg1.Type == token.Number && len(ins.arg1.Value) > 1 && ins.arg1.Value[0] == '#':
		return parseInstructionImmediateAddressing(ins)

	case ins.arg1.Value == "#" && (next1.Type == token.Identifier || next1.Type == token.Number):
		return parseInstructionImmediateAddressingWithToken(parser, ins, next1)

	case ins.arg1.Type == token.Number:
		return parseInstructionNumberParameter(ins)

	case ins.arg1.Type == token.Identifier || ins.instruction.HasAddressing(m65816.AccumulatorAddressing) || ins.arg1.Type.IsTerminator():
		return parseInstructionSingleIdentifier(parser, ins)

	default:
		return nil, fmt.Errorf("unsupported instruction argument type %s", ins.arg1.Type)
	}
}

// parseInstructionBrackets handles [dp] indirect long addressing.
func parseInstructionBrackets(parser arch.Parser, ins *instruction) (ast.Node, error) {
	parser.AdvanceReadPosition(2)

	for {
		next := parser.NextToken(0)
		switch next.Type {
		case token.EOF, token.EOL:
			return nil, errMissingParameter

		case token.RightBracket:
			next = parser.NextToken(1)
			if next.Type != token.Comma {
				return parseInstructionBracketSingle(ins)
			}

			parser.AdvanceReadPosition(2)
			ins.arg2 = parser.NextToken(0)
			return parseInstructionBracketIndexed(ins)

		default:
			return nil, fmt.Errorf("unexpected bracket token type %s", next.Type)
		}
	}
}

func parseInstructionBracketSingle(ins *instruction) (ast.Node, error) {
	argument, err := resolveArgument(ins.arg1)
	if err != nil {
		return nil, err
	}

	// [dp] -> DirectPageIndirectLongAddressing, [abs] -> AbsoluteIndirectLongAddressing
	if ins.instruction.HasAddressing(m65816.AbsoluteIndirectLongAddressing) {
		return ast.NewInstruction(ins.instruction.Name, int(m65816.AbsoluteIndirectLongAddressing), argument, ins.modifiers), nil
	}
	if ins.instruction.HasAddressing(m65816.DirectPageIndirectLongAddressing) {
		return ast.NewInstruction(ins.instruction.Name, int(m65816.DirectPageIndirectLongAddressing), argument, ins.modifiers), nil
	}

	return nil, errors.New("instruction does not support indirect long addressing")
}

func parseInstructionBracketIndexed(ins *instruction) (ast.Node, error) {
	if ins.arg2.Value != "y" && ins.arg2.Value != "Y" {
		return nil, fmt.Errorf("invalid bracket indexed register '%s'", ins.arg2.Value)
	}

	argument, err := resolveArgument(ins.arg1)
	if err != nil {
		return nil, err
	}

	if ins.instruction.HasAddressing(m65816.DirectPageIndirectLongIndexedYAddressing) {
		return ast.NewInstruction(ins.instruction.Name, int(m65816.DirectPageIndirectLongIndexedYAddressing), argument, ins.modifiers), nil
	}

	return nil, errors.New("instruction does not support indirect long indexed Y addressing")
}

func parseInstructionParentheses(parser arch.Parser, ins *instruction) (ast.Node, error) {
	parser.AdvanceReadPosition(2)

	for {
		next := parser.NextToken(0)
		switch next.Type {
		case token.EOF, token.EOL:
			return nil, errMissingParameter

		case token.Comma:
			ins.arg2 = parser.NextToken(1)
			parser.AdvanceReadPosition(2)
			return parseInstructionParenComma(parser, ins)

		case token.RightParentheses:
			next = parser.NextToken(1)
			if next.Type != token.Comma {
				return parseInstructionParenSingle(parser, ins)
			}

			parser.AdvanceReadPosition(2)
			ins.arg2 = parser.NextToken(0)
			return parseInstructionSecondIdentifier(ins, true)

		default:
			return nil, fmt.Errorf("unexpected parentheses token type %s", next.Type)
		}
	}
}

// parseInstructionParenSingle handles (arg) without following comma.
func parseInstructionParenSingle(parser arch.Parser, ins *instruction) (ast.Node, error) {
	// Check for direct page indirect or absolute indirect addressing
	if ins.instruction.HasAddressing(m65816.DirectPageIndirectAddressing) ||
		ins.instruction.HasAddressing(m65816.AbsoluteIndirectAddressing) {

		argument, err := resolveArgument(ins.arg1)
		if err != nil {
			return nil, err
		}

		if ins.instruction.HasAddressing(m65816.DirectPageIndirectAddressing) {
			return ast.NewInstruction(ins.instruction.Name, int(m65816.DirectPageIndirectAddressing), argument, ins.modifiers), nil
		}
		return ast.NewInstruction(ins.instruction.Name, int(m65816.AbsoluteIndirectAddressing), argument, ins.modifiers), nil
	}

	return parseInstructionSingleIdentifier(parser, ins)
}

// parseInstructionParenComma handles (arg,X) and (sr,S) patterns.
func parseInstructionParenComma(parser arch.Parser, ins *instruction) (ast.Node, error) {
	arg2Val := ins.arg2.Value

	// (dp,X) -> indirect X addressing
	if arg2Val == "x" || arg2Val == "X" {
		return parseInstructionSecondIdentifier(ins, true)
	}

	// (sr,S) -> check for ),Y suffix for StackRelativeIndirectIndexedY
	if arg2Val == "s" || arg2Val == "S" {
		return parseStackRelativeParentheses(parser, ins)
	}

	return nil, fmt.Errorf("invalid parenthesized second argument '%s'", arg2Val)
}

// parseStackRelativeParentheses handles (sr,S) and (sr,S),Y addressing.
func parseStackRelativeParentheses(parser arch.Parser, ins *instruction) (ast.Node, error) {
	argument, err := resolveArgument(ins.arg1)
	if err != nil {
		return nil, err
	}

	// Expect )
	next := parser.NextToken(0)
	if next.Type != token.RightParentheses {
		return nil, fmt.Errorf("expected ')' after S, got %s", next.Type)
	}

	// Check for ,Y
	next = parser.NextToken(1)
	if next.Type != token.Comma {
		// (sr,S) only — stack relative addressing
		if ins.instruction.HasAddressing(m65816.StackRelativeAddressing) {
			return ast.NewInstruction(ins.instruction.Name, int(m65816.StackRelativeAddressing), argument, ins.modifiers), nil
		}
		return nil, errors.New("instruction does not support stack relative addressing")
	}

	parser.AdvanceReadPosition(2)
	yReg := parser.NextToken(0)
	if yReg.Value != "y" && yReg.Value != "Y" {
		return nil, fmt.Errorf("expected Y register after (sr,S), got '%s'", yReg.Value)
	}

	if ins.instruction.HasAddressing(m65816.StackRelativeIndirectIndexedYAddressing) {
		return ast.NewInstruction(ins.instruction.Name, int(m65816.StackRelativeIndirectIndexedYAddressing), argument, ins.modifiers), nil
	}

	return nil, errors.New("instruction does not support stack relative indirect indexed Y addressing")
}

func parseInstructionSingleIdentifier(parser arch.Parser, ins *instruction) (ast.Node, error) {
	if m65816.BranchingInstructions.Contains(ins.instruction.Name) {
		return parseBranchingInstruction(parser, ins)
	}

	if ins.instruction.HasAddressing(m65816.AccumulatorAddressing) {
		if node := parseInstructionSingleIdentifierAccumulator(parser, ins); node != nil {
			return node, nil
		}
	}

	var addressing m65816.AddressingMode
	switch ins.addressingSize {
	case addressingAbsolute:
		if !ins.instruction.HasAddressing(m65816.AbsoluteAddressing) {
			return nil, errors.New("invalid absolute addressing mode usage")
		}
		addressing = m65816.AbsoluteAddressing

	case addressingDirectPage:
		if !ins.instruction.HasAddressing(m65816.DirectPageAddressing) {
			return nil, errors.New("invalid direct page addressing mode usage")
		}
		addressing = m65816.DirectPageAddressing

	case addressingLong:
		if !ins.instruction.HasAddressing(m65816.AbsoluteLongAddressing) {
			return nil, errors.New("invalid long addressing mode usage")
		}
		addressing = m65816.AbsoluteLongAddressing

	case addressingDefault:
		addressing = resolveDefaultAddressing(ins)
	}

	l := ast.NewLabel(ins.arg1.Value)
	return ast.NewInstruction(ins.instruction.Name, int(addressing), l, ins.modifiers), nil
}

func resolveDefaultAddressing(ins *instruction) m65816.AddressingMode {
	hasAbsolute := ins.instruction.HasAddressing(m65816.AbsoluteAddressing)
	hasDirectPage := ins.instruction.HasAddressing(m65816.DirectPageAddressing)

	switch {
	case hasAbsolute && hasDirectPage:
		return AbsoluteDirectPageAddressing
	case hasAbsolute:
		return m65816.AbsoluteAddressing
	case hasDirectPage:
		return m65816.DirectPageAddressing
	default:
		return m65816.NoAddressing
	}
}

func parseInstructionSingleIdentifierAccumulator(parser arch.Parser, ins *instruction) ast.Node {
	var usesAccumulator bool

	switch {
	case ins.arg1.Type == token.Identifier:
		if strings.ToLower(ins.arg1.Value) == "a" {
			usesAccumulator = true

			arg2 := parser.NextToken(1)
			if arg2.Type == token.Colon {
				parser.AdvanceReadPosition(-1)
			}
		}

	case ins.arg2.Type == token.Colon:

	default:
		usesAccumulator = true
	}

	if !usesAccumulator {
		return nil
	}
	return ast.NewInstruction(ins.instruction.Name, int(m65816.AccumulatorAddressing), nil, ins.modifiers)
}

func parseBranchingInstruction(parser arch.Parser, ins *instruction) (ast.Node, error) {
	var addressing m65816.AddressingMode

	switch {
	case ins.instruction.HasAddressing(m65816.RelativeAddressing):
		addressing = m65816.RelativeAddressing
	case ins.instruction.HasAddressing(m65816.RelativeLongAddressing):
		addressing = m65816.RelativeLongAddressing
	case ins.instruction.HasAddressing(m65816.AbsoluteAddressing):
		addressing = m65816.AbsoluteAddressing
	case ins.instruction.HasAddressing(m65816.AbsoluteLongAddressing):
		addressing = m65816.AbsoluteLongAddressing
	default:
		return nil, fmt.Errorf("instruction %s has no supported branching addressing mode", ins.instruction.Name)
	}

	if ins.arg1.Type == token.LeftParentheses {
		return parseBranchingIndirect(parser, ins)
	}

	if ins.arg1.Type == token.LeftBracket {
		return parseBranchingIndirectLong(parser, ins)
	}

	l := ast.NewLabel(ins.arg1.Value)
	return ast.NewInstruction(ins.instruction.Name, int(addressing), l, nil), nil
}

func parseBranchingIndirect(parser arch.Parser, ins *instruction) (ast.Node, error) {
	param := parser.NextToken(2)
	ins.arg1 = parser.NextToken(1)

	if param.Type == token.Comma {
		// (abs,X)
		xReg := parser.NextToken(2)
		if xReg.Value != "x" && xReg.Value != "X" {
			return nil, errors.New("expected X register in indirect indexed addressing")
		}
		closeParen := parser.NextToken(3)
		if closeParen.Type != token.RightParentheses {
			return nil, errors.New("missing right parentheses")
		}
		parser.AdvanceReadPosition(4)

		if !ins.instruction.HasAddressing(m65816.AbsoluteIndexedXIndirectAddressing) {
			return nil, errors.New("instruction does not support absolute indexed X indirect addressing")
		}

		l := ast.NewLabel(ins.arg1.Value)
		return ast.NewInstruction(ins.instruction.Name, int(m65816.AbsoluteIndexedXIndirectAddressing), l, nil), nil
	}

	if param.Type != token.RightParentheses {
		return nil, errors.New("missing right parentheses argument")
	}
	parser.AdvanceReadPosition(2)

	if !ins.instruction.HasAddressing(m65816.AbsoluteIndirectAddressing) {
		return nil, errors.New("instruction does not support indirect addressing")
	}

	l := ast.NewLabel(ins.arg1.Value)
	return ast.NewInstruction(ins.instruction.Name, int(m65816.AbsoluteIndirectAddressing), l, nil), nil
}

func parseBranchingIndirectLong(parser arch.Parser, ins *instruction) (ast.Node, error) {
	ins.arg1 = parser.NextToken(1)
	closeBracket := parser.NextToken(2)
	if closeBracket.Type != token.RightBracket {
		return nil, errors.New("missing right bracket")
	}
	parser.AdvanceReadPosition(3)

	if !ins.instruction.HasAddressing(m65816.AbsoluteIndirectLongAddressing) {
		return nil, errors.New("instruction does not support absolute indirect long addressing")
	}

	l := ast.NewLabel(ins.arg1.Value)
	return ast.NewInstruction(ins.instruction.Name, int(m65816.AbsoluteIndirectLongAddressing), l, nil), nil
}

// parseBlockMove handles MVN/MVP: two bank byte operands.
// Encoding: opcode, dst_bank, src_bank (destination first in binary).
func parseBlockMove(parser arch.Parser, ins *instruction) (ast.Node, error) {
	srcArg := ins.arg1

	next := parser.NextToken(1)
	if next.Type != token.Comma {
		return nil, errors.New("block move requires two operands")
	}
	parser.AdvanceReadPosition(2)
	dstArg := parser.NextToken(0)

	src, err := number.Parse(srcArg.Value)
	if err != nil {
		return nil, fmt.Errorf("parsing block move source bank: %w", err)
	}
	dst, err := number.Parse(dstArg.Value)
	if err != nil {
		return nil, fmt.Errorf("parsing block move destination bank: %w", err)
	}

	if src > math.MaxUint8 || dst > math.MaxUint8 {
		return nil, errors.New("block move bank values must be single bytes")
	}

	// Pack as (src << 8) | dst for later extraction
	packed := (src << 8) | dst
	n := ast.NewNumber(packed)
	return ast.NewInstruction(ins.instruction.Name, int(m65816.BlockMoveAddressing), n, nil), nil
}

func parseInstructionSecondIdentifier(ins *instruction, indirectAccess bool) (ast.Node, error) {
	addressings, err := extendedAddressingParam(ins, indirectAccess)
	if err != nil {
		return nil, err
	}

	argument, err := resolveArgument(ins.arg1)
	if err != nil {
		return nil, err
	}

	availableAddressing := addressings[:0]
	for _, addressing := range addressings {
		if ins.instruction.HasAddressing(addressing) {
			availableAddressing = append(availableAddressing, addressing)
		}
	}

	var addressing m65816.AddressingMode
	switch len(availableAddressing) {
	case 1:
		addressing = addressings[0]
	case 2:
		if addressings[0] == m65816.AbsoluteIndexedXAddressing {
			addressing = XAddressing
		} else {
			addressing = YAddressing
		}
	default:
		return nil, errors.New("invalid second parameter addressing mode usage")
	}

	return ast.NewInstruction(ins.instruction.Name, int(addressing), argument, ins.modifiers), nil
}

func parseInstructionImmediateAddressing(ins *instruction) (ast.Node, error) {
	if !ins.instruction.HasAddressing(m65816.ImmediateAddressing) {
		return nil, errors.New("invalid immediate addressing mode usage")
	}

	argument, err := resolveImmediateArgument(ins.arg1.Type, ins.arg1.Value)
	if err != nil {
		return nil, err
	}
	return ast.NewInstruction(ins.instruction.Name, int(m65816.ImmediateAddressing), argument, ins.modifiers), nil
}

func parseInstructionImmediateAddressingWithToken(parser arch.Parser, ins *instruction, tok token.Token) (ast.Node, error) {
	if !ins.instruction.HasAddressing(m65816.ImmediateAddressing) {
		return nil, errors.New("invalid immediate addressing mode usage")
	}

	tokenValue := tok.Value
	if tok.Type == token.Identifier {
		tokenValue = parser.ScopeLocalLabel(tokenValue)
	}
	tokenType := tok.Type

	parser.AdvanceReadPosition(2)

	argument, err := resolveImmediateArgument(tokenType, tokenValue)
	if err != nil {
		return nil, err
	}
	return ast.NewInstruction(ins.instruction.Name, int(m65816.ImmediateAddressing), argument, ins.modifiers), nil
}

func resolveImmediateArgument(tokenType token.Type, tokenValue string) (ast.Node, error) {
	if tokenType == token.Identifier {
		return ast.NewIdentifier(tokenValue), nil
	}

	i, err := number.Parse(tokenValue)
	if err != nil {
		return nil, fmt.Errorf("parsing immediate argument '%s': %w", tokenValue, err)
	}
	if i > math.MaxUint8 {
		return nil, fmt.Errorf("immediate argument '%s' exceeds byte value", tokenValue)
	}
	return ast.NewNumber(i), nil
}

func parseInstructionNumberParameter(ins *instruction) (ast.Node, error) {
	i, err := number.Parse(ins.arg1.Value)
	if err != nil {
		return nil, fmt.Errorf("parsing number argument '%s': %w", ins.arg1.Value, err)
	}

	addressing := m65816.NoAddressing

	switch ins.addressingSize {
	case addressingDirectPage:
		if !ins.instruction.HasAddressing(m65816.DirectPageAddressing) {
			return nil, errors.New("invalid direct page addressing mode usage")
		}
		if i > math.MaxUint8 {
			return nil, errors.New("direct page address exceeds byte value")
		}
		addressing = m65816.DirectPageAddressing

	case addressingAbsolute:
		if !ins.instruction.HasAddressing(m65816.AbsoluteAddressing) {
			return nil, errors.New("invalid absolute addressing mode usage")
		}
		addressing = m65816.AbsoluteAddressing

	case addressingLong:
		if !ins.instruction.HasAddressing(m65816.AbsoluteLongAddressing) {
			return nil, errors.New("invalid long addressing mode usage")
		}
		addressing = m65816.AbsoluteLongAddressing

	case addressingDefault:
		addressing = resolveDefaultNumberAddressing(ins, i)
	}

	n := ast.NewNumber(i)
	return ast.NewInstruction(ins.instruction.Name, int(addressing), n, ins.modifiers), nil
}

func resolveDefaultNumberAddressing(ins *instruction, value uint64) m65816.AddressingMode {
	switch {
	case value <= math.MaxUint8 && ins.instruction.HasAddressing(m65816.DirectPageAddressing):
		return m65816.DirectPageAddressing
	case value <= math.MaxUint16 && ins.instruction.HasAddressing(m65816.AbsoluteAddressing):
		return m65816.AbsoluteAddressing
	case ins.instruction.HasAddressing(m65816.AbsoluteLongAddressing):
		return m65816.AbsoluteLongAddressing
	case ins.instruction.HasAddressing(m65816.AbsoluteAddressing):
		return m65816.AbsoluteAddressing
	case ins.instruction.HasAddressing(m65816.DirectPageAddressing):
		return m65816.DirectPageAddressing
	default:
		return m65816.NoAddressing
	}
}

func resolveArgument(tok token.Token) (ast.Node, error) {
	switch {
	case tok.Type == token.Number:
		i, err := number.Parse(tok.Value)
		if err != nil {
			return nil, fmt.Errorf("parsing number '%s': %w", tok.Value, err)
		}
		return ast.NewNumber(i), nil

	case tok.Type == token.Identifier:
		return ast.NewLabel(tok.Value), nil

	default:
		return nil, fmt.Errorf("unsupported argument type %s", tok.Type)
	}
}

// resolveArg1Token reads and resolves the first instruction argument token, handling
// identifier scoping, unnamed label references, and dot-local label references.
func resolveArg1Token(p arch.Parser) token.Token {
	arg := p.NextToken(0)
	if arg.Type == token.Identifier {
		arg.Value = p.ScopeLocalLabel(arg.Value)
	}
	if arg.Type == token.Colon {
		if name, ok := resolveUnnamedLabelRef(p); ok {
			return token.Token{Type: token.Identifier, Value: name}
		}
	}
	if arg.Type == token.Dot {
		if name, ok := resolveDotLocalLabelRef(p); ok {
			return token.Token{Type: token.Identifier, Value: name}
		}
	}
	return arg
}

func resolveUnnamedLabelRef(p arch.Parser) (string, bool) {
	next := p.NextToken(1)
	if next.Type != token.Plus && next.Type != token.Minus {
		return "", false
	}

	forward := next.Type == token.Plus
	level := 1

	for {
		peek := p.NextToken(1 + level)
		if (forward && peek.Type == token.Plus) || (!forward && peek.Type == token.Minus) {
			level++
		} else {
			break
		}
	}

	p.AdvanceReadPosition(level)
	name := p.ResolveUnnamedLabel(forward, level)
	return name, true
}

func resolveDotLocalLabelRef(p arch.Parser) (string, bool) {
	next := p.NextToken(1)
	if next.Type != token.Identifier {
		return "", false
	}

	name := p.ResolveDotLocalLabel(next.Value)
	if name == "" {
		return "", false
	}

	p.AdvanceReadPosition(1)
	return name, true
}
