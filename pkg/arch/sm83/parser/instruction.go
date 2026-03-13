package parser

import (
	"errors"
	"fmt"
	"slices"

	"github.com/retroenv/retroasm/pkg/arch"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/number"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	cpusm83 "github.com/retroenv/retrogolib/arch/cpu/sm83"
)

var (
	errMissingOperand          = errors.New("missing operand")
	errUnsupportedOperandToken = errors.New("unsupported operand token type")
	errNoVariantMatched        = errors.New("no variant matched")
)

// ResolvedInstruction contains the selected SM83 instruction variant and parsed operand data.
type ResolvedInstruction struct {
	Addressing     cpusm83.AddressingMode
	Instruction    *cpusm83.Instruction
	RegisterParams []cpusm83.RegisterParam
	OperandValues  []ast.Node
}

type rawOperand struct {
	token token.Token

	value       ast.Node
	register    cpusm83.RegisterParam
	indirectReg cpusm83.RegisterParam
	indirect    bool
	isHLPlus    bool
	isHLMinus   bool
	isCondition bool
}

// ParseIdentifier parses an SM83 instruction and resolves the matching instruction variant.
func ParseIdentifier(p arch.Parser, mnemonic string, variants []*cpusm83.Instruction) (ast.Node, error) {
	operands, err := parseOperands(p)
	if err != nil {
		return nil, fmt.Errorf("parsing operands: %w", err)
	}

	resolved, err := resolveInstruction(mnemonic, variants, operands)
	if err != nil {
		return nil, fmt.Errorf("resolving instruction '%s': %w", mnemonic, err)
	}

	argument := ast.NewInstructionArgument(*resolved)
	return ast.NewInstruction(mnemonic, int(resolved.Addressing), argument, nil), nil
}

func parseOperands(p arch.Parser) ([]rawOperand, error) {
	next := p.NextToken(1)
	if next.Type.IsTerminator() {
		return nil, nil
	}

	p.AdvanceReadPosition(1)

	operand1, err := parseOperand(p)
	if err != nil {
		return nil, err
	}

	if p.NextToken(1).Type != token.Comma {
		return []rawOperand{operand1}, nil
	}

	p.AdvanceReadPosition(2)
	operand2, err := parseOperand(p)
	if err != nil {
		return nil, err
	}

	return []rawOperand{operand1, operand2}, nil
}

func parseOperand(p arch.Parser) (rawOperand, error) {
	tok := p.NextToken(0)

	switch tok.Type {
	case token.Number:
		return parseNumberOperand(p, tok)
	case token.Identifier:
		return parseIdentifierOperand(p, tok)
	case token.LeftParentheses:
		return parseParenthesizedOperand(p)
	case token.EOF, token.EOL, token.Comment:
		return rawOperand{}, errMissingOperand
	default:
		return rawOperand{}, fmt.Errorf("%w: %s", errUnsupportedOperandToken, tok.Type)
	}
}

func parseNumberOperand(p arch.Parser, tok token.Token) (rawOperand, error) {
	expressionOperand, matched := parseExpressionOperand(p, tok)
	if matched {
		return expressionOperand, nil
	}
	return rawOperand{token: tok}, nil
}

func parseIdentifierOperand(p arch.Parser, tok token.Token) (rawOperand, error) {
	if reg, ok := lookupRegister(tok.Value); ok {
		if cond, condOK := lookupCondition(tok.Value); condOK {
			return rawOperand{token: tok, register: reg, isCondition: true,
				indirectReg: cond}, nil
		}
		return rawOperand{token: tok, register: reg}, nil
	}
	if cond, ok := lookupCondition(tok.Value); ok {
		return rawOperand{token: tok, register: cond, isCondition: true}, nil
	}

	expressionOperand, matched := parseExpressionOperand(p, tok)
	if matched {
		return expressionOperand, nil
	}
	return rawOperand{token: tok}, nil
}

func parseParenthesizedOperand(p arch.Parser) (rawOperand, error) {
	inner := p.NextToken(1)
	if inner.Type.IsTerminator() {
		return rawOperand{}, errMissingOperand
	}

	switch inner.Type {
	case token.Identifier:
		return parseParenthesizedIdentifier(p, inner)
	case token.Number:
		return parseParenthesizedNumber(p, inner)
	default:
		return rawOperand{}, fmt.Errorf("%w in parentheses: %s", errUnsupportedOperandToken, inner.Type)
	}
}

func parseParenthesizedIdentifier(p arch.Parser, inner token.Token) (rawOperand, error) {
	next := p.NextToken(2)

	switch next.Type {
	case token.RightParentheses:
		p.AdvanceReadPosition(2)
		return buildParenthesizedRegOrLabel(inner)

	case token.Plus:
		return parseHLPlusMinus(p, inner, true)

	case token.Minus:
		return parseHLPlusMinus(p, inner, false)

	default:
		return rawOperand{}, fmt.Errorf("unsupported parenthesized identifier form near '%s'", inner.Value)
	}
}

func buildParenthesizedRegOrLabel(inner token.Token) (rawOperand, error) {
	if indReg, ok := lookupIndirectRegister(inner.Value); ok {
		return rawOperand{indirect: true, indirectReg: indReg}, nil
	}
	if reg, ok := lookupRegister(inner.Value); ok {
		return rawOperand{indirect: true, register: reg}, nil
	}

	return rawOperand{
		indirect: true,
		value:    ast.NewLabel(inner.Value),
	}, nil
}

func parseHLPlusMinus(p arch.Parser, _ token.Token, plus bool) (rawOperand, error) {
	closing := p.NextToken(3)
	if closing.Type == token.RightParentheses {
		p.AdvanceReadPosition(3)
		return rawOperand{indirect: true, isHLPlus: plus, isHLMinus: !plus}, nil
	}

	return rawOperand{}, fmt.Errorf("expected closing parenthesis after (hl%s)", map[bool]string{true: "+", false: "-"}[plus])
}

func parseParenthesizedNumber(p arch.Parser, inner token.Token) (rawOperand, error) {
	next := p.NextToken(2)

	switch next.Type {
	case token.RightParentheses:
		p.AdvanceReadPosition(2)
		value, ok, err := parseValueOperand(inner)
		if err != nil {
			return rawOperand{}, err
		}
		if !ok {
			return rawOperand{}, fmt.Errorf("unsupported parenthesized value '%s'", inner.Value)
		}
		return rawOperand{indirect: true, value: value}, nil

	case token.Plus, token.Minus:
		return parseParenthesizedExpressionOperand(p, inner)

	default:
		return rawOperand{}, errors.New("missing closing parenthesis")
	}
}

func parseParenthesizedExpressionOperand(p arch.Parser, base token.Token) (rawOperand, error) {
	tokens, consumed := parseExpressionTokenList(p, 2, token.RightParentheses)
	if consumed == 0 {
		return rawOperand{}, errors.New("expected expression in parenthesized operand")
	}

	closingOffset := 2 + consumed
	if p.NextToken(closingOffset).Type != token.RightParentheses {
		return rawOperand{}, errors.New("missing closing parenthesis")
	}

	p.AdvanceReadPosition(closingOffset)
	return rawOperand{
		indirect: true,
		value:    ast.NewExpression(append([]token.Token{base}, tokens...)...),
	}, nil
}

func parseExpressionOperand(p arch.Parser, base token.Token) (rawOperand, bool) {
	tokens, consumed := parseExpressionTokenList(p, 1, token.Comma)
	if consumed == 0 {
		return rawOperand{}, false
	}

	p.AdvanceReadPosition(consumed)
	return rawOperand{
		value: ast.NewExpression(append([]token.Token{base}, tokens...)...),
	}, true
}

func parseExpressionTokenList(p arch.Parser, startOffset int, stopToken token.Type) ([]token.Token, int) {
	var tokens []token.Token

	for consumed := 0; ; consumed++ {
		tok := p.NextToken(startOffset + consumed)

		if tok.Type == stopToken || tok.Type.IsTerminator() {
			return tokens, len(tokens)
		}

		if !isExpressionTokenAllowed(tok.Type) {
			return tokens, len(tokens)
		}

		tokens = append(tokens, tok)
	}
}

func isExpressionTokenAllowed(tokenType token.Type) bool {
	return tokenType == token.Number ||
		tokenType == token.Identifier ||
		tokenType == token.LeftParentheses ||
		tokenType == token.RightParentheses ||
		tokenType.IsOperator()
}

func parseValueOperand(tok token.Token) (ast.Node, bool, error) {
	switch tok.Type {
	case token.Number:
		value, err := number.Parse(tok.Value)
		if err != nil {
			return nil, false, fmt.Errorf("parsing number '%s': %w", tok.Value, err)
		}
		return ast.NewNumber(value), true, nil
	case token.Identifier:
		return ast.NewLabel(tok.Value), true, nil
	default:
		return nil, false, nil
	}
}

// resolveInstruction dispatches to the appropriate resolver based on operand count.
func resolveInstruction(name string, variants []*cpusm83.Instruction, operands []rawOperand) (*ResolvedInstruction, error) {
	switch len(operands) {
	case 0:
		return resolveNoOperand(variants)
	case 1:
		return resolveSingleOperand(name, variants, operands[0])
	case 2:
		return resolveTwoOperands(name, variants, operands[0], operands[1])
	default:
		return nil, fmt.Errorf("%w: expected at most 2 operands, got %d", errNoVariantMatched, len(operands))
	}
}

func resolveNoOperand(variants []*cpusm83.Instruction) (*ResolvedInstruction, error) {
	for _, variant := range variants {
		if !variant.HasAddressing(cpusm83.ImpliedAddressing) {
			continue
		}
		if len(variant.RegisterOpcodes) > 0 || len(variant.RegisterPairOpcodes) > 0 {
			continue
		}

		return &ResolvedInstruction{
			Addressing:  cpusm83.ImpliedAddressing,
			Instruction: variant,
		}, nil
	}

	for _, variant := range variants {
		if !variant.HasAddressing(cpusm83.ImpliedAddressing) {
			continue
		}

		return &ResolvedInstruction{
			Addressing:  cpusm83.ImpliedAddressing,
			Instruction: variant,
		}, nil
	}

	return nil, fmt.Errorf("%w: no implied-operand variant", errNoVariantMatched)
}

func resolveSingleOperand(name string, variants []*cpusm83.Instruction, op rawOperand) (*ResolvedInstruction, error) {
	if result := resolveSingleRegister(variants, op); result != nil {
		return result, nil
	}

	if result := resolveSingleIndirect(variants, op); result != nil {
		return result, nil
	}

	if result := resolveSingleValue(variants, op); result != nil {
		return result, nil
	}

	return nil, fmt.Errorf("%w: no single-operand variant for '%s'", errNoVariantMatched, name)
}

func resolveSingleRegister(variants []*cpusm83.Instruction, op rawOperand) *ResolvedInstruction {
	if op.register == cpusm83.RegNone || op.indirect {
		return nil
	}

	reg := op.register
	if op.isCondition && op.indirectReg != cpusm83.RegNone {
		// "c" could be register or condition — try condition first for RET C, then register.
		if result := matchConditionSingle(variants, op.indirectReg); result != nil {
			return result
		}
		// Fall through to try as register C.
	} else if op.isCondition {
		if result := matchConditionSingle(variants, reg); result != nil {
			return result
		}
		return nil
	}

	for _, variant := range variants {
		if len(variant.RegisterOpcodes) == 0 {
			continue
		}

		if _, ok := variant.RegisterOpcodes[reg]; !ok {
			continue
		}

		addressing := selectRegisterAddressing(variant)
		return &ResolvedInstruction{
			Addressing:     addressing,
			Instruction:    variant,
			RegisterParams: []cpusm83.RegisterParam{reg},
		}
	}

	return nil
}

func matchConditionSingle(variants []*cpusm83.Instruction, cond cpusm83.RegisterParam) *ResolvedInstruction {
	for _, variant := range variants {
		if len(variant.RegisterOpcodes) == 0 {
			continue
		}
		if _, ok := variant.RegisterOpcodes[cond]; !ok {
			continue
		}
		if !isCondition(cond) {
			continue
		}

		addressing := selectRegisterAddressing(variant)
		return &ResolvedInstruction{
			Addressing:     addressing,
			Instruction:    variant,
			RegisterParams: []cpusm83.RegisterParam{cond},
		}
	}
	return nil
}

func resolveSingleIndirect(variants []*cpusm83.Instruction, op rawOperand) *ResolvedInstruction {
	if !op.indirect || op.indirectReg == cpusm83.RegNone {
		return nil
	}

	if result := matchIndirectRegisterOpcode(variants, op.indirectReg); result != nil {
		return result
	}

	// Fallback: Addressing-only (e.g., JP (HL)).
	return matchIndirectAddressingOnly(variants)
}

func matchIndirectRegisterOpcode(variants []*cpusm83.Instruction, reg cpusm83.RegisterParam) *ResolvedInstruction {
	for _, variant := range variants {
		if len(variant.RegisterOpcodes) == 0 {
			continue
		}
		if _, ok := variant.RegisterOpcodes[reg]; !ok {
			continue
		}
		return &ResolvedInstruction{
			Addressing:     cpusm83.RegisterIndirectAddressing,
			Instruction:    variant,
			RegisterParams: []cpusm83.RegisterParam{reg},
		}
	}
	return nil
}

func matchIndirectAddressingOnly(variants []*cpusm83.Instruction) *ResolvedInstruction {
	for _, variant := range variants {
		if !variant.HasAddressing(cpusm83.RegisterIndirectAddressing) {
			continue
		}
		if len(variant.RegisterOpcodes) != 0 {
			continue
		}
		return &ResolvedInstruction{
			Addressing:  cpusm83.RegisterIndirectAddressing,
			Instruction: variant,
		}
	}
	return nil
}

func resolveSingleValue(variants []*cpusm83.Instruction, op rawOperand) *ResolvedInstruction {
	value, ok, err := operandValue(op)
	if err != nil || !ok {
		return nil
	}

	// Check RST vector.
	if numberValue, numberOK := value.(ast.Number); numberOK {
		if rstParam, rstOK := lookupRstVector(numberValue.Value); rstOK {
			if result := matchRstVariant(variants, rstParam); result != nil {
				return result
			}
		}
	}

	// Try value addressing (relative, extended, immediate).
	for _, variant := range variants {
		if len(variant.RegisterOpcodes) > 0 || len(variant.RegisterPairOpcodes) > 0 {
			continue
		}

		addressing, addressingOK := selectValueAddressing(variant)
		if !addressingOK {
			continue
		}

		return &ResolvedInstruction{
			Addressing:    addressing,
			Instruction:   variant,
			OperandValues: []ast.Node{value},
		}
	}

	// Second pass: allow variants with register opcodes (e.g., SUB n).
	for _, variant := range variants {
		addressing, addressingOK := selectValueAddressing(variant)
		if !addressingOK {
			continue
		}

		return &ResolvedInstruction{
			Addressing:    addressing,
			Instruction:   variant,
			OperandValues: []ast.Node{value},
		}
	}

	return nil
}

func matchRstVariant(variants []*cpusm83.Instruction, rstParam cpusm83.RegisterParam) *ResolvedInstruction {
	for _, variant := range variants {
		if len(variant.RegisterOpcodes) == 0 {
			continue
		}
		if _, ok := variant.RegisterOpcodes[rstParam]; !ok {
			continue
		}

		return &ResolvedInstruction{
			Addressing:     cpusm83.ImpliedAddressing,
			Instruction:    variant,
			RegisterParams: []cpusm83.RegisterParam{rstParam},
		}
	}
	return nil
}

func resolveTwoOperands(name string, variants []*cpusm83.Instruction, op1, op2 rawOperand) (*ResolvedInstruction, error) {
	if result := resolveSpecialLD(variants, op1, op2); result != nil {
		return result, nil
	}
	if result := resolveRegisterPair(variants, op1, op2); result != nil {
		return result, nil
	}
	if result := resolveIndirectLoadStore(variants, op1, op2); result != nil {
		return result, nil
	}
	if result := resolveIndirectImmediate(variants, op1, op2); result != nil {
		return result, nil
	}
	if result := resolveAluRegisterPair(variants, op1, op2); result != nil {
		return result, nil
	}

	return resolveTwoOperandsFallback(name, variants, op1, op2)
}

func resolveTwoOperandsFallback(
	name string,
	variants []*cpusm83.Instruction,
	op1, op2 rawOperand,
) (*ResolvedInstruction, error) {

	if result := resolveExtendedMemory(variants, op1, op2); result != nil {
		return result, nil
	}
	if result := resolveConditionAddress(variants, op1, op2); result != nil {
		return result, nil
	}
	if result := resolveRegisterValue(variants, op1, op2); result != nil {
		return result, nil
	}
	if result := resolveBitRegister(variants, op1, op2); result != nil {
		return result, nil
	}

	return nil, fmt.Errorf("%w: no two-operand variant for '%s'", errNoVariantMatched, name)
}

func resolveSpecialLD(variants []*cpusm83.Instruction, op1, op2 rawOperand) *ResolvedInstruction {
	if result := resolveSpecialLDAccumulator(variants, op1, op2); result != nil {
		return result
	}
	if result := resolveSpecialLDIndirectStore(variants, op1, op2); result != nil {
		return result
	}

	// SP, HL — LD SP,HL
	if op1.register == cpusm83.RegSP && op2.register == cpusm83.RegHL && !op1.indirect && !op2.indirect {
		return matchSpecialImplied(variants, cpusm83.LdSPHL)
	}

	// HL, SP+e — LD HL,SP+e
	if op1.register == cpusm83.RegHL && !op1.indirect && !op2.indirect {
		if result := resolveHLSPOffset(variants, op2); result != nil {
			return result
		}
	}

	return nil
}

func resolveSpecialLDAccumulator(variants []*cpusm83.Instruction, op1, op2 rawOperand) *ResolvedInstruction {
	if op1.register != cpusm83.RegA || !op2.indirect {
		return nil
	}
	if op2.isHLPlus {
		return matchSpecialImplied(variants, cpusm83.LdAHLPlus)
	}
	if op2.isHLMinus {
		return matchSpecialImplied(variants, cpusm83.LdAHLMinus)
	}
	if op2.register == cpusm83.RegC {
		return matchSpecialImplied(variants, cpusm83.LdAC)
	}
	return nil
}

func resolveSpecialLDIndirectStore(variants []*cpusm83.Instruction, op1, op2 rawOperand) *ResolvedInstruction {
	if !op1.indirect || op2.register != cpusm83.RegA {
		return nil
	}
	if op1.isHLPlus {
		return matchSpecialImplied(variants, cpusm83.LdHLPlusA)
	}
	if op1.isHLMinus {
		return matchSpecialImplied(variants, cpusm83.LdHLMinusA)
	}
	if op1.register == cpusm83.RegC {
		return matchSpecialImplied(variants, cpusm83.LdCA)
	}
	return nil
}

func resolveHLSPOffset(variants []*cpusm83.Instruction, op2 rawOperand) *ResolvedInstruction {
	// Check if second operand starts with "sp" identifier followed by value tokens.
	if op2.register != cpusm83.RegSP {
		return nil
	}
	// LD HL,SP+e is resolved when sp is the second operand with a value.
	// The parser sees "sp" as a register, so we need a value following it.
	// This case is handled when the operand is an expression involving SP.
	// For now, match LdHLSPOffset when it's in variants and op2 has a value.
	if op2.value == nil {
		return nil
	}
	return matchSpecialWithValue(variants, cpusm83.LdHLSPOffset, op2.value)
}

func matchSpecialImplied(variants []*cpusm83.Instruction, target *cpusm83.Instruction) *ResolvedInstruction {
	if !slices.Contains(variants, target) {
		return nil
	}

	return &ResolvedInstruction{
		Addressing:  cpusm83.ImpliedAddressing,
		Instruction: target,
	}
}

func matchSpecialWithValue(variants []*cpusm83.Instruction, target *cpusm83.Instruction, value ast.Node) *ResolvedInstruction {
	if !slices.Contains(variants, target) {
		return nil
	}

	return &ResolvedInstruction{
		Addressing:    cpusm83.ImmediateAddressing,
		Instruction:   target,
		OperandValues: []ast.Node{value},
	}
}

func resolveRegisterPair(variants []*cpusm83.Instruction, op1, op2 rawOperand) *ResolvedInstruction {
	if op1.indirect || op2.indirect {
		return nil
	}
	if op1.register == cpusm83.RegNone || op2.register == cpusm83.RegNone {
		return nil
	}

	reg1 := op1.register
	reg2 := op2.register

	for _, variant := range variants {
		if len(variant.RegisterPairOpcodes) == 0 {
			continue
		}

		key := [2]cpusm83.RegisterParam{reg1, reg2}
		if _, ok := variant.RegisterPairOpcodes[key]; !ok {
			continue
		}

		addressing := cpusm83.RegisterAddressing
		if variant.HasAddressing(cpusm83.RegisterIndirectAddressing) {
			addressing = cpusm83.RegisterIndirectAddressing
		}

		return &ResolvedInstruction{
			Addressing:     addressing,
			Instruction:    variant,
			RegisterParams: []cpusm83.RegisterParam{reg1, reg2},
		}
	}

	return nil
}

func resolveIndirectLoadStore(variants []*cpusm83.Instruction, op1, op2 rawOperand) *ResolvedInstruction {
	var regOp, indOp rawOperand
	var isLoad bool
	switch {
	case op2.indirect && !op1.indirect:
		regOp, indOp = op1, op2
		isLoad = true
	case op1.indirect && !op2.indirect:
		regOp, indOp = op2, op1
		isLoad = false
	default:
		return nil
	}

	if indOp.indirectReg == cpusm83.RegNone {
		return nil
	}
	if regOp.register == cpusm83.RegNone {
		return nil
	}

	// Build candidate keys for RegisterOpcodes.
	keys := indirectRegisterKeys(regOp.register, indOp.indirectReg, isLoad)
	if result := matchIndirectKeys(variants, keys); result != nil {
		return result
	}

	// Try RegisterPairOpcodes (e.g., LD (HL),A uses pair keys).
	pairKeys := indirectPairKeys(regOp.register, indOp.indirectReg, isLoad)
	return matchIndirectPairKeys(variants, pairKeys)
}

func indirectRegisterKeys(reg, ind cpusm83.RegisterParam, isLoad bool) []cpusm83.RegisterParam {
	var keys []cpusm83.RegisterParam

	if isLoad {
		if mapped, ok := hlLoadRegisterParam(reg, ind); ok {
			keys = append(keys, mapped)
		}
	} else {
		keys = append(keys, ind)
		keys = append(keys, reg)
	}

	return keys
}

func hlLoadRegisterParam(reg, ind cpusm83.RegisterParam) (cpusm83.RegisterParam, bool) {
	if ind == cpusm83.RegHLIndirect {
		switch reg {
		case cpusm83.RegB:
			return cpusm83.RegLoadHLB, true
		case cpusm83.RegC:
			return cpusm83.RegLoadHLC, true
		case cpusm83.RegD:
			return cpusm83.RegLoadHLD, true
		case cpusm83.RegE:
			return cpusm83.RegLoadHLE, true
		case cpusm83.RegH:
			return cpusm83.RegLoadHLH, true
		case cpusm83.RegL:
			return cpusm83.RegLoadHLL, true
		case cpusm83.RegA:
			return cpusm83.RegLoadHLA, true
		}
	}
	if ind == cpusm83.RegBCIndirect && reg == cpusm83.RegA {
		return cpusm83.RegLoadBC, true
	}
	if ind == cpusm83.RegDEIndirect && reg == cpusm83.RegA {
		return cpusm83.RegLoadDE, true
	}

	return cpusm83.RegNone, false
}

func indirectPairKeys(reg, ind cpusm83.RegisterParam, isLoad bool) [][2]cpusm83.RegisterParam {
	var keys [][2]cpusm83.RegisterParam
	if isLoad {
		keys = append(keys, [2]cpusm83.RegisterParam{reg, ind})
	} else {
		keys = append(keys, [2]cpusm83.RegisterParam{ind, reg})
	}
	return keys
}

func matchIndirectKeys(variants []*cpusm83.Instruction, keys []cpusm83.RegisterParam) *ResolvedInstruction {
	for _, variant := range variants {
		if len(variant.RegisterOpcodes) == 0 {
			continue
		}

		for _, key := range keys {
			if _, ok := variant.RegisterOpcodes[key]; !ok {
				continue
			}

			addressing := indirectAddressing(variant)
			return &ResolvedInstruction{
				Addressing:     addressing,
				Instruction:    variant,
				RegisterParams: []cpusm83.RegisterParam{key},
			}
		}
	}
	return nil
}

func matchIndirectPairKeys(variants []*cpusm83.Instruction, keys [][2]cpusm83.RegisterParam) *ResolvedInstruction {
	for _, variant := range variants {
		if len(variant.RegisterPairOpcodes) == 0 {
			continue
		}

		for _, key := range keys {
			if _, ok := variant.RegisterPairOpcodes[key]; !ok {
				continue
			}

			addressing := indirectAddressing(variant)
			return &ResolvedInstruction{
				Addressing:     addressing,
				Instruction:    variant,
				RegisterParams: []cpusm83.RegisterParam{key[0], key[1]},
			}
		}
	}
	return nil
}

func indirectAddressing(variant *cpusm83.Instruction) cpusm83.AddressingMode {
	if variant.HasAddressing(cpusm83.RegisterIndirectAddressing) {
		return cpusm83.RegisterIndirectAddressing
	}
	if variant.HasAddressing(cpusm83.RegisterAddressing) {
		return cpusm83.RegisterAddressing
	}
	return cpusm83.RegisterIndirectAddressing
}

func resolveIndirectImmediate(variants []*cpusm83.Instruction, op1, op2 rawOperand) *ResolvedInstruction {
	if !op1.indirect || op1.indirectReg == cpusm83.RegNone {
		return nil
	}
	if op2.register != cpusm83.RegNone {
		return nil
	}

	value, ok, err := operandValue(op2)
	if err != nil || !ok {
		return nil
	}

	// LD (HL),n — RegisterIndirectAddressing without RegisterOpcodes.
	for _, variant := range variants {
		if len(variant.RegisterOpcodes) != 0 {
			continue
		}
		if !variant.HasAddressing(cpusm83.RegisterIndirectAddressing) {
			continue
		}

		return &ResolvedInstruction{
			Addressing:    cpusm83.RegisterIndirectAddressing,
			Instruction:   variant,
			OperandValues: []ast.Node{value},
		}
	}

	return nil
}

func resolveAluRegisterPair(variants []*cpusm83.Instruction, op1, op2 rawOperand) *ResolvedInstruction {
	if op1.indirect || op2.indirect {
		return nil
	}
	if op1.register == cpusm83.RegNone || op2.register == cpusm83.RegNone {
		return nil
	}

	for _, variant := range variants {
		if len(variant.RegisterOpcodes) == 0 || variant.Name == cpusm83.LdName {
			continue
		}

		if _, ok := variant.RegisterOpcodes[op2.register]; !ok {
			continue
		}

		addressing := selectRegisterAddressing(variant)
		return &ResolvedInstruction{
			Addressing:     addressing,
			Instruction:    variant,
			RegisterParams: []cpusm83.RegisterParam{op2.register},
		}
	}

	return nil
}

func resolveExtendedMemory(variants []*cpusm83.Instruction, op1, op2 rawOperand) *ResolvedInstruction {
	// Register, (nn) — load from address.
	if !op1.indirect && op2.indirect && op2.value != nil && op2.indirectReg == cpusm83.RegNone {
		for _, variant := range variants {
			if !variant.HasAddressing(cpusm83.ExtendedAddressing) {
				continue
			}
			if len(variant.RegisterOpcodes) != 0 {
				continue
			}

			return &ResolvedInstruction{
				Addressing:    cpusm83.ExtendedAddressing,
				Instruction:   variant,
				OperandValues: []ast.Node{op2.value},
			}
		}
	}

	// (nn), Register — store to address.
	if op1.indirect && op1.value != nil && op1.indirectReg == cpusm83.RegNone && !op2.indirect {
		for _, variant := range variants {
			if !variant.HasAddressing(cpusm83.ExtendedAddressing) {
				continue
			}
			if len(variant.RegisterOpcodes) != 0 {
				continue
			}

			return &ResolvedInstruction{
				Addressing:    cpusm83.ExtendedAddressing,
				Instruction:   variant,
				OperandValues: []ast.Node{op1.value},
			}
		}
	}

	return nil
}

func resolveConditionAddress(variants []*cpusm83.Instruction, op1, op2 rawOperand) *ResolvedInstruction {
	if !op1.isCondition {
		return nil
	}

	value, ok, err := operandValue(op2)
	if err != nil || !ok {
		return nil
	}

	cond := op1.register
	// For "c" as condition, use RegCondC.
	if op1.indirectReg != cpusm83.RegNone && isCondition(op1.indirectReg) {
		cond = op1.indirectReg
	}

	for _, variant := range variants {
		if len(variant.RegisterOpcodes) == 0 {
			continue
		}
		if _, condOK := variant.RegisterOpcodes[cond]; !condOK {
			continue
		}
		if !isCondition(cond) {
			continue
		}

		addressing, addressingOK := selectValueAddressing(variant)
		if !addressingOK {
			continue
		}

		return &ResolvedInstruction{
			Addressing:     addressing,
			Instruction:    variant,
			RegisterParams: []cpusm83.RegisterParam{cond},
			OperandValues:  []ast.Node{value},
		}
	}

	return nil
}

func resolveRegisterValue(variants []*cpusm83.Instruction, op1, op2 rawOperand) *ResolvedInstruction {
	if op1.register == cpusm83.RegNone || op1.indirect {
		return nil
	}

	value, ok, err := operandValue(op2)
	if err != nil || !ok {
		return nil
	}

	reg := op1.register

	for _, variant := range variants {
		addressing, addressingOK := selectValueAddressing(variant)
		if !addressingOK {
			continue
		}
		if op2.indirect && addressing == cpusm83.ImmediateAddressing {
			continue
		}

		if _, regOK := variant.RegisterOpcodes[reg]; !regOK {
			continue
		}

		var regParams []cpusm83.RegisterParam
		if reg != cpusm83.RegA || addressing != cpusm83.ImmediateAddressing || variant.Name == cpusm83.LdName {
			regParams = []cpusm83.RegisterParam{reg}
		}

		return &ResolvedInstruction{
			Addressing:     addressing,
			Instruction:    variant,
			RegisterParams: regParams,
			OperandValues:  []ast.Node{value},
		}
	}

	return nil
}

func resolveBitRegister(variants []*cpusm83.Instruction, op1, op2 rawOperand) *ResolvedInstruction {
	value, ok, err := operandValue(op1)
	if err != nil || !ok {
		return nil
	}

	reg := op2.register
	if op2.indirect && op2.indirectReg != cpusm83.RegNone {
		reg = op2.indirectReg
	}
	if reg == cpusm83.RegNone {
		return nil
	}

	for _, variant := range variants {
		if !variant.HasAddressing(cpusm83.RegisterAddressing) {
			continue
		}
		if !isCBBitInstruction(variant) {
			continue
		}

		return &ResolvedInstruction{
			Addressing:     cpusm83.BitAddressing,
			Instruction:    variant,
			RegisterParams: []cpusm83.RegisterParam{reg},
			OperandValues:  []ast.Node{value},
		}
	}

	return nil
}

func isCBBitInstruction(instruction *cpusm83.Instruction) bool {
	return instruction == cpusm83.CBBit || instruction == cpusm83.CBRes || instruction == cpusm83.CBSet
}

func selectValueAddressing(variant *cpusm83.Instruction) (cpusm83.AddressingMode, bool) {
	switch {
	case variant.HasAddressing(cpusm83.RelativeAddressing):
		return cpusm83.RelativeAddressing, true
	case variant.HasAddressing(cpusm83.ExtendedAddressing):
		return cpusm83.ExtendedAddressing, true
	case variant.HasAddressing(cpusm83.ImmediateAddressing):
		return cpusm83.ImmediateAddressing, true
	default:
		return cpusm83.NoAddressing, false
	}
}

func selectRegisterAddressing(variant *cpusm83.Instruction) cpusm83.AddressingMode {
	if variant.HasAddressing(cpusm83.RegisterAddressing) {
		return cpusm83.RegisterAddressing
	}
	if variant.HasAddressing(cpusm83.ImpliedAddressing) {
		return cpusm83.ImpliedAddressing
	}

	if len(variant.Addressing) == 1 {
		for addressing := range variant.Addressing {
			return addressing
		}
	}

	return cpusm83.NoAddressing
}

func operandValue(op rawOperand) (ast.Node, bool, error) {
	if op.value != nil {
		return op.value, true, nil
	}

	return parseValueOperand(op.token)
}
