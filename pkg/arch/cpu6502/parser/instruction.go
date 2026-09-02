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
	"github.com/retroenv/retrogolib/arch/cpu/cpu6502"
)

var errMissingParameter = errors.New("missing parameter")

// ParseIdentifier parses an instruction identifier and returns an AST node.
func ParseIdentifier(parser arch.Parser, ins *cpu6502.Instruction) (ast.Node, error) {
	if len(ins.Addressing) == 1 && ins.HasAddressing(cpu6502.ImpliedAddressing) {
		return newInstruction(ins, int(cpu6502.ImpliedAddressing), nil, nil), nil
	}

	node, err := parseInstruction(parser, ins)
	if err != nil {
		return nil, fmt.Errorf("parsing instruction %s: %w", ins.Name, err)
	}
	return node, nil
}

type instruction struct {
	instruction    *cpu6502.Instruction
	addressingSize addressingSize
	modifiers      []ast.Modifier
	arg1           token.Token
	arg2           token.Token
}

// newInstruction creates an unresolved CPU6502 AST instruction. The
// architecture-bound parser/codec applies the scoped opcode identity.
func newInstruction(ins *cpu6502.Instruction, addressing int, arg ast.Node, modifiers []ast.Modifier) ast.Instruction {
	return ast.NewInstruction(ins.Name, addressing, arg, modifiers)
}

func parseInstruction(parser arch.Parser, instructionDetails *cpu6502.Instruction) (ast.Node, error) {
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
		parser.AdvanceReadPosition(2)
		ins.arg2 = resolveArg1Token(parser)
		if ins.instruction.HasAddressing(cpu6502.ZeroPageRelativeAddressing) {
			return parseZeroPageRelativeInstruction(ins)
		}
		return parseInstructionSecondIdentifier(ins, false)
	}

	switch {
	case ins.arg1.Type == token.LeftParentheses:
		ins.arg1 = next1
		return parseInstructionParentheses(parser, ins)

	case ins.arg1.Type == token.Number && len(ins.arg1.Value) > 1 && ins.arg1.Value[0] == '#':
		// Handle immediate numbers that are tokenized as a single token: LDA #32
		return parseInstructionImmediateAddressing(ins)

	case ins.arg1.Value == "#" && next1.Type == token.LeftParentheses:
		// Handle immediate addressing with parenthesized expression: LDA #(LABEL-1)
		return parseInstructionImmediateAddressingWithExpression(parser, ins)

	case ins.arg1.Value == "#" && (next1.Type == token.Identifier || next1.Type == token.Number):
		// Handle immediate addressing with separate tokens: LDA #MAX_ENTITIES or LDA #$FF
		return parseInstructionImmediateAddressingWithToken(parser, ins, next1)

	case ins.arg1.Type == token.Number:
		return parseInstructionNumber(parser, ins)

	case ins.arg1.Type == token.Identifier || ins.instruction.HasAddressing(cpu6502.AccumulatorAddressing) || ins.arg1.Type.IsTerminator():
		return parseInstructionSingleIdentifier(parser, ins)

	default:
		return nil, fmt.Errorf("unsupported instruction argument type %s", ins.arg1.Type)
	}
}

func parseInstructionNumber(parser arch.Parser, ins *instruction) (ast.Node, error) {
	if _, ok := cpu6502.BranchingInstructions[ins.instruction.Name]; ok {
		return parseBranchingInstruction(parser, ins)
	}
	return parseInstructionNumberParameter(ins)
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
				addressing, ok := parenthesizedAddressing(ins)
				if ok {
					argument, err := argumentFromToken(ins.arg1)
					if err != nil {
						return nil, err
					}
					return newInstruction(
						ins.instruction,
						int(addressing),
						argument,
						ins.modifiers,
					), nil
				}
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

func parenthesizedAddressing(ins *instruction) (cpu6502.AddressingMode, bool) {
	switch ins.addressingSize {
	case addressingZeroPage:
		return cpu6502.ZeroPageIndirectAddressing,
			ins.instruction.HasAddressing(cpu6502.ZeroPageIndirectAddressing)
	case addressingAbsolute:
		return cpu6502.IndirectAddressing, ins.instruction.HasAddressing(cpu6502.IndirectAddressing)
	default:
		if ins.instruction.HasAddressing(cpu6502.ZeroPageIndirectAddressing) {
			return cpu6502.ZeroPageIndirectAddressing, true
		}
		return cpu6502.IndirectAddressing, ins.instruction.HasAddressing(cpu6502.IndirectAddressing)
	}
}

func parseZeroPageRelativeInstruction(ins *instruction) (ast.Node, error) {
	if ins.addressingSize == addressingAbsolute {
		return nil, errors.New("zero-page-relative instruction cannot use absolute addressing")
	}
	if len(ins.modifiers) != 0 {
		return nil, errors.New("zero-page-relative instruction does not support modifiers")
	}
	zeroPage, err := argumentFromToken(ins.arg1)
	if err != nil {
		return nil, fmt.Errorf("parsing zero-page operand: %w", err)
	}
	if value, ok := ast.NumberValue(zeroPage); ok && value > math.MaxUint8 {
		return nil, errors.New("zeropage address exceeds byte value")
	}
	target, err := argumentFromToken(ins.arg2)
	if err != nil {
		return nil, fmt.Errorf("parsing relative target: %w", err)
	}
	arguments := ast.NewInstructionArguments(zeroPage, target)
	return newInstruction(ins.instruction, int(cpu6502.ZeroPageRelativeAddressing), arguments, nil), nil
}

func parseInstructionSingleIdentifier(parser arch.Parser, ins *instruction) (ast.Node, error) {
	if _, ok := cpu6502.BranchingInstructions[ins.instruction.Name]; ok {
		return parseBranchingInstruction(parser, ins)
	}

	if ins.instruction.HasAddressing(cpu6502.AccumulatorAddressing) {
		if node := parseInstructionSingleIdentifierAccumulator(parser, ins); node != nil {
			return node, nil
		}
	}

	var addressing cpu6502.AddressingMode
	switch ins.addressingSize {
	case addressingAbsolute:
		if !ins.instruction.HasAddressing(cpu6502.AbsoluteAddressing) {
			return nil, errors.New("invalid absolute addressing mode usage")
		}
		addressing = cpu6502.AbsoluteAddressing

	case addressingZeroPage:
		if !ins.instruction.HasAddressing(cpu6502.ZeroPageAddressing) {
			return nil, errors.New("invalid zeropage addressing mode usage")
		}
		addressing = cpu6502.ZeroPageAddressing

	case addressingDefault:
		// Use ambiguous mode - will be resolved during address assignment
		hasAbsolute := ins.instruction.HasAddressing(cpu6502.AbsoluteAddressing)
		hasZeroPage := ins.instruction.HasAddressing(cpu6502.ZeroPageAddressing)

		switch {
		case hasAbsolute && hasZeroPage:
			addressing = AbsoluteZeroPageAddressing
		case hasAbsolute:
			addressing = cpu6502.AbsoluteAddressing
		case hasZeroPage:
			addressing = cpu6502.ZeroPageAddressing
		default:
			return nil, errors.New("instruction has no absolute or zeropage addressing modes")
		}
	}

	l := ast.NewLabel(ins.arg1.Value)
	return newInstruction(ins.instruction, int(addressing), l, ins.modifiers), nil
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
	return newInstruction(ins.instruction, int(cpu6502.AccumulatorAddressing), nil, ins.modifiers)
}

func parseBranchingInstruction(parser arch.Parser, ins *instruction) (ast.Node, error) {
	addressing := cpu6502.RelativeAddressing
	if !ins.instruction.HasAddressing(cpu6502.RelativeAddressing) {
		addressing = cpu6502.AbsoluteAddressing
	}

	if ins.arg1.Type == token.LeftParentheses {
		param := parser.NextToken(2)
		if param.Type != token.RightParentheses {
			return nil, errors.New("missing right parentheses argument")
		}
		ins.arg1 = parser.NextToken(1)

		if !ins.instruction.HasAddressing(cpu6502.IndirectAddressing) {
			return nil, errors.New("instruction does not support indirect addressing")
		}

		addressing = cpu6502.IndirectAddressing
		parser.AdvanceReadPosition(2)
	}

	argument, err := argumentFromToken(ins.arg1)
	if err != nil {
		return nil, err
	}
	return newInstruction(ins.instruction, int(addressing), argument, nil), nil
}

func argumentFromToken(argument token.Token) (ast.Node, error) {
	switch argument.Type {
	case token.Number:
		value, err := number.Parse(argument.Value)
		if err != nil {
			return nil, fmt.Errorf("parsing number %q: %w", argument.Value, err)
		}
		return ast.NewNumber(value), nil
	case token.Identifier:
		return ast.NewLabel(argument.Value), nil
	default:
		return nil, fmt.Errorf("unsupported argument type %s", argument.Type)
	}
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

	var addressing cpu6502.AddressingMode
	switch len(availableAddressing) {
	case 1:
		addressing = availableAddressing[0]
	case 2:
		switch availableAddressing[0] {
		case cpu6502.AbsoluteXAddressing:
			addressing = XAddressing
		case cpu6502.AbsoluteYAddressing:
			addressing = YAddressing
		default:
			return nil, errors.New("indirect addressing size is ambiguous")
		}
	default:
		return nil, errors.New("invalid second parameter addressing mode usage")
	}

	return newInstruction(ins.instruction, int(addressing), argument, ins.modifiers), nil
}

func parseInstructionImmediateAddressing(ins *instruction) (ast.Node, error) {
	if !ins.instruction.HasAddressing(cpu6502.ImmediateAddressing) {
		return nil, errors.New("invalid immediate addressing mode usage")
	}

	argument, err := resolveImmediateArgument(ins.arg1.Type, ins.arg1.Value)
	if err != nil {
		return nil, err
	}
	return newInstruction(ins.instruction, int(cpu6502.ImmediateAddressing), argument, ins.modifiers), nil
}

func parseInstructionImmediateAddressingWithToken(parser arch.Parser, ins *instruction, tok token.Token) (ast.Node, error) {
	if !ins.instruction.HasAddressing(cpu6502.ImmediateAddressing) {
		return nil, errors.New("invalid immediate addressing mode usage")
	}

	// Save the token value before advancing, in case advancing affects the token.
	tokenValue := tok.Value
	if tok.Type == token.Identifier {
		tokenValue = parser.ScopeLocalLabel(tokenValue)
	}
	tokenType := tok.Type

	parser.AdvanceReadPosition(2) // Skip past # and the token

	argument, err := resolveImmediateArgument(tokenType, tokenValue)
	if err != nil {
		return nil, err
	}
	return newInstruction(ins.instruction, int(cpu6502.ImmediateAddressing), argument, ins.modifiers), nil
}

func parseInstructionImmediateAddressingWithExpression(parser arch.Parser, ins *instruction) (ast.Node, error) {
	if !ins.instruction.HasAddressing(cpu6502.ImmediateAddressing) {
		return nil, errors.New("invalid immediate addressing mode usage")
	}

	// Collect tokens starting from '(' (offset 1 from '#') until the balanced ')'
	var tokens []token.Token
	depth := 0

	for offset := 1; ; offset++ {
		tok := parser.NextToken(offset)

		switch tok.Type {
		case token.EOF, token.EOL:
			return nil, errors.New("unexpected end of immediate expression")

		case token.LeftParentheses:
			depth++
			tokens = append(tokens, tok)

		case token.RightParentheses:
			depth--
			tokens = append(tokens, tok)
			if depth == 0 {
				parser.AdvanceReadPosition(offset + 1)
				argument := ast.NewExpression(tokens...)
				return newInstruction(ins.instruction, int(cpu6502.ImmediateAddressing), argument, ins.modifiers), nil
			}

		case token.Identifier:
			t := tok
			t.Value = parser.ScopeLocalLabel(t.Value)
			tokens = append(tokens, t)

		case token.Number:
			tokens = append(tokens, tok)

		default:
			if tok.Type.IsOperator() {
				tokens = append(tokens, tok)
				break
			}
			return nil, fmt.Errorf("unexpected token '%s' in immediate expression", tok.Type)
		}
	}
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

	addressing := cpu6502.NoAddressing

	switch ins.addressingSize {
	case addressingZeroPage:
		if !ins.instruction.HasAddressing(cpu6502.ZeroPageAddressing) {
			return nil, errors.New("invalid zeropage addressing mode usage")
		}
		if i > math.MaxUint8 {
			return nil, errors.New("zeropage address exceeds byte value")
		}
		addressing = cpu6502.ZeroPageAddressing

	case addressingAbsolute:
		if !ins.instruction.HasAddressing(cpu6502.AbsoluteAddressing) {
			return nil, errors.New("invalid absolute addressing mode usage")
		}
		addressing = cpu6502.AbsoluteAddressing

	case addressingDefault:
		// Prefer zero page for values that fit in a byte
		switch {
		case i <= math.MaxUint8 && ins.instruction.HasAddressing(cpu6502.ZeroPageAddressing):
			addressing = cpu6502.ZeroPageAddressing
		case ins.instruction.HasAddressing(cpu6502.AbsoluteAddressing):
			addressing = cpu6502.AbsoluteAddressing
		case ins.instruction.HasAddressing(cpu6502.ZeroPageAddressing):
			addressing = cpu6502.ZeroPageAddressing
		default:
			return nil, errors.New("instruction has no absolute or zeropage addressing modes")
		}
	}

	n := ast.NewNumber(i)
	return newInstruction(ins.instruction, int(addressing), n, ins.modifiers), nil
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

// resolveUnnamedLabelRef checks if the current position has a ca65-style unnamed label reference
// (:+, :-, :++, :--, etc.) and returns the resolved synthetic label name.
func resolveUnnamedLabelRef(p arch.Parser) (string, bool) {
	next := p.NextToken(1)
	if next.Type != token.Plus && next.Type != token.Minus {
		return "", false
	}

	forward := next.Type == token.Plus
	level := 1

	// Count consecutive +/- tokens for multi-level references
	for {
		peek := p.NextToken(1 + level)
		if (forward && peek.Type == token.Plus) || (!forward && peek.Type == token.Minus) {
			level++
		} else {
			break
		}
	}

	p.AdvanceReadPosition(level) // advance past the +/- tokens (: stays as position base)
	name := p.ResolveUnnamedLabel(forward, level)
	return name, true
}

// resolveDotLocalLabelRef checks if the current Dot token starts a NESASM-style
// dot-prefixed local label reference (.label) and returns the scoped name.
func resolveDotLocalLabelRef(p arch.Parser) (string, bool) {
	next := p.NextToken(1)
	if next.Type != token.Identifier {
		return "", false
	}

	name := p.ResolveDotLocalLabel(next.Value)
	if name == "" {
		return "", false
	}

	p.AdvanceReadPosition(1) // advance past the identifier (. stays as position base)
	return name, true
}
