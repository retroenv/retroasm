package parser

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/retroenv/assembler/lexer/token"
	"github.com/retroenv/assembler/number"
	"github.com/retroenv/assembler/parser/ast"
	"github.com/retroenv/assembler/parser/directives"
	. "github.com/retroenv/retrogolib/addressing"
	"github.com/retroenv/retrogolib/cpu"
)

type instruction struct {
	instruction    *cpu.Instruction
	addressingSize addressingSize
	modifiers      []ast.Modifier
	arg1           token.Token
	arg2           token.Token
}

func (p *Parser) parseInstruction(instructionDetails *cpu.Instruction) (ast.Node, error) {
	p.readPosition++

	var err error
	ins := &instruction{
		instruction: instructionDetails,
	}

	ins.addressingSize, err = p.parseAddressSize(instructionDetails)
	if err != nil {
		return nil, fmt.Errorf("parsing addressing size: %w", err)
	}

	ins.arg1 = p.NextToken(0)
	ins.modifiers = directives.ParseModifier(p)

	next1 := p.NextToken(1)
	if next1.Type == token.Comma {
		p.readPosition += 2
		ins.arg2 = p.NextToken(0)
		return parseInstructionSecondIdentifier(ins, false)
	}

	switch {
	case ins.arg1.Type == token.LeftParentheses:
		ins.arg1 = next1
		return p.parseInstructionParentheses(ins)

	case ins.arg1.Type == token.Number && ins.arg1.Value[0] == '#':
		return parseInstructionImmediateAddressing(ins)

	case ins.arg1.Type == token.Number:
		return parseInstructionNumberParameter(ins)

	case ins.arg1.Type == token.Identifier || ins.instruction.HasAddressing(AccumulatorAddressing) || ins.arg1.Type.IsTerminator():
		return p.parseInstructionSingleIdentifier(ins)

	default:
		return nil, fmt.Errorf("unsupported instruction argument type %s", ins.arg1.Type)
	}
}

func (p *Parser) parseInstructionParentheses(ins *instruction) (ast.Node, error) {
	p.readPosition += 2

	for {
		next := p.NextToken(0)
		switch next.Type {
		case token.EOF, token.EOL:
			return nil, errMissingParameter

		case token.Comma:
			ins.arg2 = p.NextToken(1)
			p.readPosition += 2
			return parseInstructionSecondIdentifier(ins, true)

		case token.RightParentheses:
			next = p.NextToken(1)
			if next.Type != token.Comma {
				return p.parseInstructionSingleIdentifier(ins)
			}

			p.readPosition += 2
			ins.arg2 = p.NextToken(0)
			return parseInstructionSecondIdentifier(ins, true)

		default:
			return nil, fmt.Errorf("unexpected parentheses token type %s", next.Type)
		}
	}
}

func (p *Parser) parseInstructionSingleIdentifier(ins *instruction) (ast.Node, error) {
	if _, ok := p.arch.BranchingInstructions[ins.instruction.Name]; ok {
		return p.parseBranchingInstruction(ins)
	}

	if ins.instruction.HasAddressing(AccumulatorAddressing) {
		if node := p.parseInstructionSingleIdentifierAccumulator(ins); node != nil {
			return node, nil
		}
	}

	var addressing Mode
	switch {
	case ins.addressingSize != addressingZeroPage && ins.instruction.HasAddressing(AbsoluteAddressing):
		addressing = AbsoluteAddressing

	case ins.addressingSize != addressingAbsolute && ins.instruction.HasAddressing(ZeroPageAddressing):
		addressing = ZeroPageAddressing

	default:
		return nil, errors.New("invalid number addressing mode usage")
	}

	l := ast.NewLabel(ins.arg1.Value)
	return ast.NewInstruction(ins.instruction.Name, addressing, l, ins.modifiers), nil
}

func (p *Parser) parseInstructionSingleIdentifierAccumulator(ins *instruction) ast.Node {
	var usesAccumulator bool

	switch {
	case ins.arg1.Type == token.Identifier:
		if strings.ToLower(ins.arg1.Value) == "a" {
			usesAccumulator = true

			// handle the edge case of an instruction being used that supports accumulator addressing but
			// does not specify the accumulator as parameter and a label follows as the nextToken token with the
			// same name as the accumulator register a
			arg2 := p.NextToken(1)
			if arg2.Type == token.Colon {
				p.readPosition--
			}
		}

	case ins.arg2.Type == token.Colon:

	default:
		usesAccumulator = true
	}

	if !usesAccumulator {
		return nil
	}
	return ast.NewInstruction(ins.instruction.Name, AccumulatorAddressing, nil, ins.modifiers)
}

func (p *Parser) parseBranchingInstruction(ins *instruction) (ast.Node, error) {
	addressing := RelativeAddressing
	if !ins.instruction.HasAddressing(RelativeAddressing) {
		addressing = AbsoluteAddressing
	}

	if ins.arg1.Type == token.LeftParentheses {
		param := p.NextToken(2)
		if param.Type != token.RightParentheses {
			return nil, errors.New("missing right parentheses argument")
		}
		ins.arg1 = p.NextToken(1)

		if !ins.instruction.HasAddressing(IndirectAddressing) {
			return nil, errors.New("instruction does not support indirect addressing")
		}

		addressing = IndirectAddressing
		p.readPosition += 2
	}

	l := ast.NewLabel(ins.arg1.Value)
	return ast.NewInstruction(ins.instruction.Name, addressing, l, nil), nil
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

	var addressing Mode
	switch len(availableAddressing) {
	case 1:
		addressing = addressings[0]
	case 2:
		if addressings[0] == AbsoluteXAddressing {
			addressing = ast.XAddressing
		} else {
			addressing = ast.YAddressing
		}
	default:
		return nil, errors.New("invalid second parameter addressing mode usage")
	}

	return ast.NewInstruction(ins.instruction.Name, addressing, argument, ins.modifiers), nil
}

func parseInstructionImmediateAddressing(ins *instruction) (ast.Node, error) {
	if !ins.instruction.HasAddressing(ImmediateAddressing) {
		return nil, errors.New("invalid immediate addressing mode usage")
	}

	i, err := number.Parse(ins.arg1.Value)
	if err != nil {
		return nil, fmt.Errorf("parsing immediate argument '%s': %w", ins.arg1.Value, err)
	}
	if i > math.MaxUint8 {
		return nil, fmt.Errorf("immediate argument '%s' exceeds byte value", ins.arg1.Value)
	}
	n := ast.NewNumber(i)
	return ast.NewInstruction(ins.instruction.Name, ImmediateAddressing, n, ins.modifiers), nil
}

func parseInstructionNumberParameter(ins *instruction) (ast.Node, error) {
	i, err := number.Parse(ins.arg1.Value)
	if err != nil {
		return nil, fmt.Errorf("parsing number argument '%s': %w", ins.arg1.Value, err)
	}

	addressing := NoAddressing

	switch ins.addressingSize {
	case addressingZeroPage:
		if !ins.instruction.HasAddressing(ZeroPageAddressing) {
			return nil, errors.New("invalid zeropage addressing mode usage")
		}
		if i > math.MaxUint8 {
			return nil, errors.New("zeropage address exceeds byte value")
		}
		addressing = ZeroPageAddressing

	case addressingAbsolute:
		if !ins.instruction.HasAddressing(AbsoluteAddressing) {
			return nil, errors.New("invalid absolute addressing mode usage")
		}
		addressing = AbsoluteAddressing

	case addressingDefault:
		if ins.instruction.HasAddressing(AbsoluteAddressing) {
			addressing = AbsoluteAddressing
		} else {
			if !ins.instruction.HasAddressing(ZeroPageAddressing) {
				return nil, errors.New("instruction has no absolute or zeropage addressing modes")
			}
			addressing = ZeroPageAddressing
		}
	}

	n := ast.NewNumber(i)
	return ast.NewInstruction(ins.instruction.Name, addressing, n, ins.modifiers), nil
}
