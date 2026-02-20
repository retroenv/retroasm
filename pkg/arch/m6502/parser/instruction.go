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
	"github.com/retroenv/retrogolib/arch/cpu/m6502"
)

type instruction struct {
	instruction    *m6502.Instruction
	addressingSize addressingSize
	modifiers      []ast.Modifier
	arg1           token.Token
	arg2           token.Token
}

var errMissingParameter = errors.New("missing parameter")

func ParseIdentifier(parser arch.Parser, ins *m6502.Instruction) (ast.Node, error) {
	if len(ins.Addressing) == 1 && ins.HasAddressing(m6502.ImpliedAddressing) {
		return ast.NewInstruction(ins.Name, int(m6502.ImpliedAddressing), nil, nil), nil
	}

	node, err := parseInstruction(parser, ins)
	if err != nil {
		return nil, fmt.Errorf("parsing instruction %s: %w", ins.Name, err)
	}
	return node, nil
}

func parseInstruction(parser arch.Parser, instructionDetails *m6502.Instruction) (ast.Node, error) {
	parser.AdvanceReadPosition(1)

	var err error
	ins := &instruction{
		instruction: instructionDetails,
	}

	ins.addressingSize, err = parseAddressSize(parser, instructionDetails)
	if err != nil {
		return nil, fmt.Errorf("parsing addressing size: %w", err)
	}

	ins.arg1 = parser.NextToken(0)
	ins.modifiers = directives.ParseModifier(parser)

	next1 := parser.NextToken(1)
	if next1.Type == token.Comma {
		parser.AdvanceReadPosition(2)
		ins.arg2 = parser.NextToken(0)
		return parseInstructionSecondIdentifier(ins, false)
	}

	switch {
	case ins.arg1.Type == token.LeftParentheses:
		ins.arg1 = next1
		return parseInstructionParentheses(parser, ins)

	case ins.arg1.Type == token.Number && len(ins.arg1.Value) > 1 && ins.arg1.Value[0] == '#':
		// Handle immediate numbers that are tokenized as a single token: LDA #32
		return parseInstructionImmediateAddressing(ins)

	case ins.arg1.Value == "#" && (next1.Type == token.Identifier || next1.Type == token.Number):
		// Handle immediate addressing with separate tokens: LDA #MAX_ENTITIES or LDA #$FF
		return parseInstructionImmediateAddressingWithToken(parser, ins, next1)

	case ins.arg1.Type == token.Number:
		return parseInstructionNumberParameter(ins)

	case ins.arg1.Type == token.Identifier || ins.instruction.HasAddressing(m6502.AccumulatorAddressing) || ins.arg1.Type.IsTerminator():
		return parseInstructionSingleIdentifier(parser, ins)

	default:
		return nil, fmt.Errorf("unsupported instruction argument type %s", ins.arg1.Type)
	}
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
			return parseInstructionSecondIdentifier(ins, true)

		case token.RightParentheses:
			next = parser.NextToken(1)
			if next.Type != token.Comma {
				return parseInstructionSingleIdentifier(parser, ins)
			}

			parser.AdvanceReadPosition(2)
			ins.arg2 = parser.NextToken(0)
			return parseInstructionSecondIdentifier(ins, true)

		default:
			return nil, fmt.Errorf("unexpected parentheses token type %s", next.Type)
		}
	}
}

func parseInstructionSingleIdentifier(parser arch.Parser, ins *instruction) (ast.Node, error) {
	if _, ok := m6502.BranchingInstructions[ins.instruction.Name]; ok {
		return parseBranchingInstruction(parser, ins)
	}

	if ins.instruction.HasAddressing(m6502.AccumulatorAddressing) {
		if node := parseInstructionSingleIdentifierAccumulator(parser, ins); node != nil {
			return node, nil
		}
	}

	var addressing m6502.AddressingMode
	switch ins.addressingSize {
	case addressingAbsolute:
		if !ins.instruction.HasAddressing(m6502.AbsoluteAddressing) {
			return nil, errors.New("invalid absolute addressing mode usage")
		}
		addressing = m6502.AbsoluteAddressing

	case addressingZeroPage:
		if !ins.instruction.HasAddressing(m6502.ZeroPageAddressing) {
			return nil, errors.New("invalid zeropage addressing mode usage")
		}
		addressing = m6502.ZeroPageAddressing

	case addressingDefault:
		// Use ambiguous mode - will be resolved during address assignment
		hasAbsolute := ins.instruction.HasAddressing(m6502.AbsoluteAddressing)
		hasZeroPage := ins.instruction.HasAddressing(m6502.ZeroPageAddressing)

		switch {
		case hasAbsolute && hasZeroPage:
			addressing = AbsoluteZeroPageAddressing
		case hasAbsolute:
			addressing = m6502.AbsoluteAddressing
		case hasZeroPage:
			addressing = m6502.ZeroPageAddressing
		default:
			return nil, errors.New("instruction has no absolute or zeropage addressing modes")
		}
	}

	l := ast.NewLabel(ins.arg1.Value)
	return ast.NewInstruction(ins.instruction.Name, int(addressing), l, ins.modifiers), nil
}

func parseInstructionSingleIdentifierAccumulator(parser arch.Parser, ins *instruction) ast.Node {
	var usesAccumulator bool

	switch {
	case ins.arg1.Type == token.Identifier:
		if strings.ToLower(ins.arg1.Value) == "a" {
			usesAccumulator = true

			// handle the edge case of an instruction being used that supports accumulator addressing but
			// does not specify the accumulator as parameter and a label follows as the nextToken token with the
			// same name as the accumulator register a
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
	return ast.NewInstruction(ins.instruction.Name, int(m6502.AccumulatorAddressing), nil, ins.modifiers)
}

func parseBranchingInstruction(parser arch.Parser, ins *instruction) (ast.Node, error) {
	addressing := m6502.RelativeAddressing
	if !ins.instruction.HasAddressing(m6502.RelativeAddressing) {
		addressing = m6502.AbsoluteAddressing
	}

	if ins.arg1.Type == token.LeftParentheses {
		param := parser.NextToken(2)
		if param.Type != token.RightParentheses {
			return nil, errors.New("missing right parentheses argument")
		}
		ins.arg1 = parser.NextToken(1)

		if !ins.instruction.HasAddressing(m6502.IndirectAddressing) {
			return nil, errors.New("instruction does not support indirect addressing")
		}

		addressing = m6502.IndirectAddressing
		parser.AdvanceReadPosition(2)
	}

	l := ast.NewLabel(ins.arg1.Value)
	return ast.NewInstruction(ins.instruction.Name, int(addressing), l, nil), nil
}

func parseInstructionSecondIdentifier(ins *instruction, indirectAccess bool) (ast.Node, error) {
	addressings, err := extendedAddressingParam(ins, indirectAccess)
	if err != nil {
		return nil, err
	}

	var argument ast.Node

	switch {
	case ins.arg1.Type == token.Number:
		i, err := number.Parse(ins.arg1.Value)
		if err != nil {
			return nil, fmt.Errorf("parsing number '%s': %w", ins.arg1.Value, err)
		}
		argument = ast.NewNumber(i)

	case ins.arg1.Type == token.Identifier:
		argument = ast.NewLabel(ins.arg1.Value)

	default:
		return nil, fmt.Errorf("unsupported argument type %s", ins.arg1.Type)
	}

	availableAddressing := addressings[:0]
	for _, addressing := range addressings {
		if ins.instruction.HasAddressing(addressing) {
			availableAddressing = append(availableAddressing, addressing)
		}
	}

	var addressing m6502.AddressingMode
	switch len(availableAddressing) {
	case 1:
		addressing = addressings[0]
	case 2:
		if addressings[0] == m6502.AbsoluteXAddressing {
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
	if !ins.instruction.HasAddressing(m6502.ImmediateAddressing) {
		return nil, errors.New("invalid immediate addressing mode usage")
	}

	argument, err := resolveImmediateArgument(ins.arg1.Type, ins.arg1.Value)
	if err != nil {
		return nil, err
	}
	return ast.NewInstruction(ins.instruction.Name, int(m6502.ImmediateAddressing), argument, ins.modifiers), nil
}

func parseInstructionImmediateAddressingWithToken(parser arch.Parser, ins *instruction, tok token.Token) (ast.Node, error) {
	if !ins.instruction.HasAddressing(m6502.ImmediateAddressing) {
		return nil, errors.New("invalid immediate addressing mode usage")
	}

	// Save the token value before advancing, in case advancing affects the token.
	tokenValue := tok.Value
	tokenType := tok.Type

	parser.AdvanceReadPosition(2) // Skip past # and the token

	argument, err := resolveImmediateArgument(tokenType, tokenValue)
	if err != nil {
		return nil, err
	}
	return ast.NewInstruction(ins.instruction.Name, int(m6502.ImmediateAddressing), argument, ins.modifiers), nil
}

// resolveImmediateArgument parses an immediate addressing argument, returning
// an identifier node for constant references or a number node for literals.
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

	addressing := m6502.NoAddressing

	switch ins.addressingSize {
	case addressingZeroPage:
		if !ins.instruction.HasAddressing(m6502.ZeroPageAddressing) {
			return nil, errors.New("invalid zeropage addressing mode usage")
		}
		if i > math.MaxUint8 {
			return nil, errors.New("zeropage address exceeds byte value")
		}
		addressing = m6502.ZeroPageAddressing

	case addressingAbsolute:
		if !ins.instruction.HasAddressing(m6502.AbsoluteAddressing) {
			return nil, errors.New("invalid absolute addressing mode usage")
		}
		addressing = m6502.AbsoluteAddressing

	case addressingDefault:
		// Prefer zero page for values that fit in a byte
		switch {
		case i <= math.MaxUint8 && ins.instruction.HasAddressing(m6502.ZeroPageAddressing):
			addressing = m6502.ZeroPageAddressing
		case ins.instruction.HasAddressing(m6502.AbsoluteAddressing):
			addressing = m6502.AbsoluteAddressing
		case ins.instruction.HasAddressing(m6502.ZeroPageAddressing):
			addressing = m6502.ZeroPageAddressing
		default:
			return nil, errors.New("instruction has no absolute or zeropage addressing modes")
		}
	}

	n := ast.NewNumber(i)
	return ast.NewInstruction(ins.instruction.Name, int(addressing), n, ins.modifiers), nil
}
