// Package parser implements the Chip-8 instruction parser.
package parser

import (
	"errors"
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/arch"
	"github.com/retroenv/retroasm/lexer/token"
	"github.com/retroenv/retroasm/number"
	"github.com/retroenv/retroasm/parser/ast"
	"github.com/retroenv/retrogolib/arch/cpu/chip8"
)

// ParseIdentifier parses a Chip-8 instruction and returns the corresponding AST node.
func ParseIdentifier(parser arch.Parser, ins *chip8.Instruction) (ast.Node, error) {
	// Handle implied addressing (no operands)
	if len(ins.Addressing) == 1 && hasAddressing(ins, chip8.ImpliedAddressing) {
		return ast.NewInstruction(ins.Name, int(chip8.ImpliedAddressing), nil, nil), nil
	}

	node, err := parseInstruction(parser, ins)
	if err != nil {
		return nil, fmt.Errorf("parsing instruction %s: %w", ins.Name, err)
	}
	return node, nil
}

func parseInstruction(parser arch.Parser, ins *chip8.Instruction) (ast.Node, error) {
	parser.AdvanceReadPosition(1)

	arg1 := parser.NextToken(0)

	// Check for terminator (instruction with no args or implied addressing)
	if arg1.Type.IsTerminator() {
		if hasAddressing(ins, chip8.ImpliedAddressing) {
			return ast.NewInstruction(ins.Name, int(chip8.ImpliedAddressing), nil, nil), nil
		}
		return nil, errors.New("missing instruction argument")
	}

	// Check for comma (multiple arguments)
	next1 := parser.NextToken(1)
	if next1.Type == token.Comma {
		parser.AdvanceReadPosition(2)
		arg2 := parser.NextToken(0)
		return parseInstructionTwoArgs(ins, arg1, arg2, parser)
	}

	// Single argument
	return parseInstructionSingleArg(ins, arg1)
}

// nolint: gocritic
func parseInstructionSingleArg(ins *chip8.Instruction, arg1 token.Token) (ast.Node, error) {
	switch {
	case arg1.Type == token.Number:
		// Absolute address (JP addr, CALL addr, LD I, addr)
		i, err := number.Parse(arg1.Value)
		if err != nil {
			return nil, fmt.Errorf("parsing number '%s': %w", arg1.Value, err)
		}
		if i > 0xFFF {
			return nil, fmt.Errorf("address %d exceeds 12-bit range", i)
		}

		// Determine addressing mode
		var addressing chip8.Mode
		if hasAddressing(ins, chip8.AbsoluteAddressing) {
			addressing = chip8.AbsoluteAddressing
		} else if hasAddressing(ins, chip8.IAbsoluteAddressing) {
			addressing = chip8.IAbsoluteAddressing
		} else {
			return nil, errors.New("invalid absolute addressing mode usage")
		}

		n := ast.NewNumber(i)
		return ast.NewInstruction(ins.Name, int(addressing), n, nil), nil

	case arg1.Type == token.Identifier:
		// Label reference
		label := arg1.Value

		// Determine addressing mode
		var addressing chip8.Mode
		if hasAddressing(ins, chip8.AbsoluteAddressing) {
			addressing = chip8.AbsoluteAddressing
		} else if hasAddressing(ins, chip8.IAbsoluteAddressing) {
			addressing = chip8.IAbsoluteAddressing
		} else {
			return nil, errors.New("invalid label addressing mode usage")
		}

		l := ast.NewLabel(label)
		return ast.NewInstruction(ins.Name, int(addressing), l, nil), nil

	default:
		return nil, fmt.Errorf("unsupported instruction argument type %s", arg1.Type)
	}
}

// nolint: gocognit, gocritic, gocyclo, cyclop, funlen, maintidx
func parseInstructionTwoArgs(ins *chip8.Instruction, arg1, arg2 token.Token, parser arch.Parser) (ast.Node, error) {
	// Parse register arguments (Vx, Vy, etc.)
	reg1, isReg1 := parseRegister(arg1)
	reg2, isReg2 := parseRegister(arg2)

	// Special registers
	isV0 := isReg1 && reg1 == 0 && strings.ToLower(arg1.Value) == "v0"
	isDT := arg1.Type == token.Identifier && strings.ToLower(arg1.Value) == "dt"
	isST := arg1.Type == token.Identifier && strings.ToLower(arg1.Value) == "st"
	isI := arg1.Type == token.Identifier && strings.ToLower(arg1.Value) == "i"
	isF := arg1.Type == token.Identifier && strings.ToLower(arg1.Value) == "f"
	isB := arg1.Type == token.Identifier && strings.ToLower(arg1.Value) == "b"

	isDT2 := arg2.Type == token.Identifier && strings.ToLower(arg2.Value) == "dt"
	isK2 := arg2.Type == token.Identifier && strings.ToLower(arg2.Value) == "k"

	// Check for indirect addressing [I]
	isIndirect1 := arg1.Type == token.LeftBracket
	isIndirect2 := arg2.Type == token.LeftBracket

	if isIndirect1 || isIndirect2 {
		return parseIndirectAddressing(ins, arg1, arg2, parser)
	}

	// Handle various two-argument addressing modes
	switch {
	case isReg1 && isReg2:
		// Vx, Vy - Register-register addressing
		if !hasAddressing(ins, chip8.RegisterRegisterAddressing) {
			return nil, errors.New("instruction does not support register-register addressing")
		}
		// Encode both registers in the argument
		value := (uint64(reg1) << 4) | uint64(reg2)
		n := ast.NewNumber(value)
		return ast.NewInstruction(ins.Name, int(chip8.RegisterRegisterAddressing), n, nil), nil

	case isReg1 && arg2.Type == token.Number:
		// Vx, byte - Register-value addressing
		val, err := number.Parse(arg2.Value)
		if err != nil {
			return nil, fmt.Errorf("parsing value '%s': %w", arg2.Value, err)
		}
		if val > 0xFF {
			return nil, fmt.Errorf("value %d exceeds byte range", val)
		}
		if !hasAddressing(ins, chip8.RegisterValueAddressing) {
			return nil, errors.New("instruction does not support register-value addressing")
		}
		// Encode register and value
		value := (uint64(reg1) << 8) | val
		n := ast.NewNumber(value)
		return ast.NewInstruction(ins.Name, int(chip8.RegisterValueAddressing), n, nil), nil

	case isV0 && arg2.Type == token.Number:
		// V0, addr - V0 + absolute addressing (JP V0, addr)
		val, err := number.Parse(arg2.Value)
		if err != nil {
			return nil, fmt.Errorf("parsing address '%s': %w", arg2.Value, err)
		}
		if val > 0xFFF {
			return nil, fmt.Errorf("address %d exceeds 12-bit range", val)
		}
		if !hasAddressing(ins, chip8.V0AbsoluteAddressing) {
			return nil, errors.New("instruction does not support V0 + absolute addressing")
		}
		n := ast.NewNumber(val)
		return ast.NewInstruction(ins.Name, int(chip8.V0AbsoluteAddressing), n, nil), nil

	case isI && arg2.Type == token.Number:
		// I, addr - I register addressing
		val, err := number.Parse(arg2.Value)
		if err != nil {
			return nil, fmt.Errorf("parsing address '%s': %w", arg2.Value, err)
		}
		if val > 0xFFF {
			return nil, fmt.Errorf("address %d exceeds 12-bit range", val)
		}
		if !hasAddressing(ins, chip8.IAbsoluteAddressing) {
			return nil, errors.New("instruction does not support I + absolute addressing")
		}
		n := ast.NewNumber(val)
		return ast.NewInstruction(ins.Name, int(chip8.IAbsoluteAddressing), n, nil), nil

	case isReg1 && isDT2:
		// Vx, DT - Load from delay timer
		if !hasAddressing(ins, chip8.RegisterDTAddressing) {
			return nil, errors.New("instruction does not support Vx, DT addressing")
		}
		n := ast.NewNumber(uint64(reg1))
		return ast.NewInstruction(ins.Name, int(chip8.RegisterDTAddressing), n, nil), nil

	case isReg1 && isK2:
		// Vx, K - Wait for key press
		if !hasAddressing(ins, chip8.RegisterKAddressing) {
			return nil, errors.New("instruction does not support Vx, K addressing")
		}
		n := ast.NewNumber(uint64(reg1))
		return ast.NewInstruction(ins.Name, int(chip8.RegisterKAddressing), n, nil), nil

	case isDT && isReg2:
		// DT, Vx - Store to delay timer
		if !hasAddressing(ins, chip8.DTRegisterAddressing) {
			return nil, errors.New("instruction does not support DT, Vx addressing")
		}
		n := ast.NewNumber(uint64(reg2))
		return ast.NewInstruction(ins.Name, int(chip8.DTRegisterAddressing), n, nil), nil

	case isST && isReg2:
		// ST, Vx - Store to sound timer
		if !hasAddressing(ins, chip8.STRegisterAddressing) {
			return nil, errors.New("instruction does not support ST, Vx addressing")
		}
		n := ast.NewNumber(uint64(reg2))
		return ast.NewInstruction(ins.Name, int(chip8.STRegisterAddressing), n, nil), nil

	case isF && isReg2:
		// F, Vx - Load font sprite
		if !hasAddressing(ins, chip8.FRegisterAddressing) {
			return nil, errors.New("instruction does not support F, Vx addressing")
		}
		n := ast.NewNumber(uint64(reg2))
		return ast.NewInstruction(ins.Name, int(chip8.FRegisterAddressing), n, nil), nil

	case isB && isReg2:
		// B, Vx - BCD representation
		if !hasAddressing(ins, chip8.BRegisterAddressing) {
			return nil, errors.New("instruction does not support B, Vx addressing")
		}
		n := ast.NewNumber(uint64(reg2))
		return ast.NewInstruction(ins.Name, int(chip8.BRegisterAddressing), n, nil), nil

	case isI && isReg2:
		// I, Vx - Add Vx to I
		if !hasAddressing(ins, chip8.IRegisterAddressing) {
			return nil, errors.New("instruction does not support ADD I, Vx addressing")
		}
		n := ast.NewNumber(uint64(reg2))
		return ast.NewInstruction(ins.Name, int(chip8.IRegisterAddressing), n, nil), nil

	default:
		// Check for three-argument instructions (DRW Vx, Vy, nibble)
		next := parser.NextToken(1)
		if next.Type == token.Comma {
			parser.AdvanceReadPosition(2)
			arg3 := parser.NextToken(0)
			return parseInstructionThreeArgs(ins, arg1, arg2, arg3)
		}
		return nil, errors.New("unsupported addressing mode")
	}
}

func parseInstructionThreeArgs(ins *chip8.Instruction, arg1, arg2, arg3 token.Token) (ast.Node, error) {
	// DRW Vx, Vy, nibble
	reg1, isReg1 := parseRegister(arg1)
	reg2, isReg2 := parseRegister(arg2)

	if !isReg1 || !isReg2 {
		return nil, errors.New("first two arguments must be registers")
	}

	if arg3.Type != token.Number {
		return nil, errors.New("third argument must be a number")
	}

	val, err := number.Parse(arg3.Value)
	if err != nil {
		return nil, fmt.Errorf("parsing nibble '%s': %w", arg3.Value, err)
	}
	if val > 0xF {
		return nil, fmt.Errorf("nibble %d exceeds 4-bit range", val)
	}

	if !hasAddressing(ins, chip8.RegisterRegisterNibbleAddressing) {
		return nil, errors.New("instruction does not support register-register-nibble addressing")
	}

	// Encode registers and nibble
	value := (uint64(reg1) << 8) | (uint64(reg2) << 4) | val
	n := ast.NewNumber(value)
	return ast.NewInstruction(ins.Name, int(chip8.RegisterRegisterNibbleAddressing), n, nil), nil
}

func parseIndirectAddressing(ins *chip8.Instruction, arg1, arg2 token.Token, parser arch.Parser) (ast.Node, error) {
	// Handle [I], Vx or Vx, [I]
	if arg1.Type == token.LeftBracket {
		// [I], Vx - Store registers to [I]
		iToken := parser.NextToken(-1)
		if iToken.Type != token.Identifier || strings.ToLower(iToken.Value) != "i" {
			return nil, errors.New("indirect addressing requires [I]")
		}

		// Skip [I] and check for ]
		parser.AdvanceReadPosition(1)
		rightBracket := parser.NextToken(0)
		if rightBracket.Type != token.RightBracket {
			return nil, errors.New("missing right bracket")
		}

		reg2, isReg2 := parseRegister(arg2)
		if !isReg2 {
			return nil, errors.New("second argument must be a register")
		}

		if !hasAddressing(ins, chip8.IIndirectRegisterAddressing) {
			return nil, errors.New("instruction does not support [I], Vx addressing")
		}

		n := ast.NewNumber(uint64(reg2))
		return ast.NewInstruction(ins.Name, int(chip8.IIndirectRegisterAddressing), n, nil), nil
	}

	if arg2.Type == token.LeftBracket {
		// Vx, [I] - Load registers from [I]
		reg1, isReg1 := parseRegister(arg1)
		if !isReg1 {
			return nil, errors.New("first argument must be a register")
		}

		parser.AdvanceReadPosition(1)
		iToken := parser.NextToken(0)
		if iToken.Type != token.Identifier || strings.ToLower(iToken.Value) != "i" {
			return nil, errors.New("indirect addressing requires [I]")
		}

		parser.AdvanceReadPosition(1)
		rightBracket := parser.NextToken(0)
		if rightBracket.Type != token.RightBracket {
			return nil, errors.New("missing right bracket")
		}

		if !hasAddressing(ins, chip8.RegisterIndirectIAddressing) {
			return nil, errors.New("instruction does not support Vx, [I] addressing")
		}

		n := ast.NewNumber(uint64(reg1))
		return ast.NewInstruction(ins.Name, int(chip8.RegisterIndirectIAddressing), n, nil), nil
	}

	return nil, errors.New("invalid indirect addressing")
}

func parseRegister(tok token.Token) (byte, bool) {
	if tok.Type != token.Identifier {
		return 0, false
	}

	val := strings.ToLower(tok.Value)
	if len(val) < 2 || val[0] != 'v' {
		return 0, false
	}

	// Parse register number (0-F)
	regStr := val[1:]
	if len(regStr) == 1 {
		c := regStr[0]
		if c >= '0' && c <= '9' {
			return c - '0', true
		}
		if c >= 'a' && c <= 'f' {
			return 10 + (c - 'a'), true
		}
	}

	return 0, false
}

func hasAddressing(ins *chip8.Instruction, mode chip8.Mode) bool {
	_, ok := ins.Addressing[mode]
	return ok
}
