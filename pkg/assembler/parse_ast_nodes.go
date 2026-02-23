package assembler

import (
	"errors"
	"fmt"
	"strings"

	"github.com/retroenv/retroasm/pkg/assembler/config"
	"github.com/retroenv/retroasm/pkg/expression"
	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/number"
	"github.com/retroenv/retroasm/pkg/parser/ast"
	"github.com/retroenv/retroasm/pkg/scope"
)

type parseAST[T any] struct {
	cfg *config.Config[T]
	// a function that reads in a file, for testing includes, defaults to os.ReadFile
	fileReader func(name string) ([]byte, error)

	currentScope   *scope.Scope // current scope, can be a function scope with file scope as parent
	currentSegment *segment     // the current segment being parsed

	segments      map[string]*segment // maps segment name to segment
	segmentsOrder []*segment          // sorted list of all parsed segments
}

//nolint:cyclop,funlen // type switch with one case per AST node type
func parseASTNode[T any](asm *parseAST[T], node ast.Node) ([]ast.Node, error) {
	var (
		err   error
		nodes []ast.Node
	)

	switch n := node.(type) {
	case ast.Data:
		nodes, err = parseData(n)

	case ast.Alias:
		nodes, err = parseAlias(asm, n)

	case ast.Label:
		nodes, err = parseLabel(asm, n)

	case ast.Function:
		nodes, err = parseFunction(asm, n)

	case ast.FunctionEnd:
		nodes, err = parseFunctionEnd(asm, n)

	case ast.Instruction:
		nodes, err = parseInstruction(n)

	case ast.Include:
		nodes, err = parseInclude(asm, n)

	case ast.Macro:
		nodes, err = parseMacro(n)

	case ast.Variable:
		parseVariable(n)

		// default case for node types that do not have special handling at this point
	default:
		return []ast.Node{n}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("parsing node type %T: %w", node, err)
	}
	return nodes, nil
}

func parseSegment[T any](asm *parseAST[T], astSegment ast.Segment) error {
	name := strings.Trim(astSegment.Name, "\"'")

	seg, ok := asm.segments[name]
	if ok {
		// do not create a segment twice
		asm.currentSegment = seg
		return nil
	}

	segmentConfig, ok := asm.cfg.Segments[name]
	if !ok {
		return fmt.Errorf("configuration for segment '%s' not found", name)
	}

	seg = &segment{
		config: segmentConfig,
		nodes:  nil,
	}
	asm.currentSegment = seg
	asm.segments[seg.config.SegmentName] = seg
	asm.segmentsOrder = append(asm.segmentsOrder, seg)
	return nil
}

func parseData(astData ast.Data) ([]ast.Node, error) {
	dat := &data{
		fill:  astData.Fill,
		width: astData.Width,
		size:  astData.Size,
	}
	if dat.size == nil {
		dat.size = expression.New()
	}

	switch astData.Type {
	case ast.AddressType:
		refType := fullAddress
		switch astData.ReferenceType {
		case ast.LowAddressByte:
			refType = lowAddressByte
			dat.width = 1
		case ast.HighAddressByte:
			refType = highAddressByte
			dat.width = 1
		}

		if err := parseDataAddress(dat, astData.Values, refType); err != nil {
			return nil, fmt.Errorf("parsing data address: %w", err)
		}

	case ast.DataType:
		dat.expression = astData.Values

	default:
		return nil, fmt.Errorf("unsupported data type %d", astData.Type)
	}

	return []ast.Node{dat}, nil
}

func parseDataAddress(dat *data, expression *expression.Expression, refType referenceType) error {
	width := dat.width
	if refType == lowAddressByte || refType == highAddressByte {
		width = 1
	}

	tokens := expression.Tokens()
	for _, tok := range tokens {
		switch tok.Type {
		case token.Identifier:
			ref := reference{
				name: tok.Value,
				typ:  refType,
			}
			dat.values = append(dat.values, ref)

		case token.Number:
			i, err := number.Parse(tok.Value)
			if err != nil {
				return fmt.Errorf("parsing number '%s': %w", tok.Value, err)
			}
			if err := number.CheckDataWidth(i, width); err != nil {
				return fmt.Errorf("checking data byte width: %w", err)
			}
			b, err := number.WriteToBytes(i, width)
			if err != nil {
				return fmt.Errorf("writing number as bytes: %w", err)
			}
			dat.values = append(dat.values, b)

		default:
			return fmt.Errorf("unsupported value type %T", tok.Type)
		}
	}

	return nil
}

func parseAlias[T any](asm *parseAST[T], alias ast.Alias) ([]ast.Node, error) {
	typ := scope.AliasType
	if !alias.SymbolReusable {
		typ = scope.EquType
	}

	sym, err := scope.NewSymbol(asm.currentScope, alias.Name, typ)
	if err != nil {
		return nil, fmt.Errorf("creating symbol: %w", err)
	}
	sym.SetExpression(alias.Expression)
	return []ast.Node{&symbol{Symbol: sym}}, nil
}

func parseLabel[T any](asm *parseAST[T], label ast.Label) ([]ast.Node, error) {
	sym, err := scope.NewSymbol(asm.currentScope, label.Name, scope.LabelType)
	if err != nil {
		return nil, fmt.Errorf("creating symbol: %w", err)
	}

	return []ast.Node{&symbol{Symbol: sym}}, nil
}

func parseInstruction(astInstruction ast.Instruction) ([]ast.Node, error) {
	ins := &instruction{
		name:       astInstruction.Name,
		addressing: astInstruction.Addressing,
		argument:   astInstruction.Argument,
	}

	switch arg := astInstruction.Argument.(type) {
	case nil:

	case ast.Number:
		value := arg.Value
		if len(astInstruction.Modifier) > 0 {
			offset, err := modifierOffset(astInstruction.Modifier)
			if err != nil {
				return nil, err
			}
			value += uint64(offset)
		}
		ins.argument = value

	case ast.Label:
		name, err := nameWithModifiers(arg.Name, astInstruction.Modifier)
		if err != nil {
			return nil, err
		}
		ins.argument = reference{name: name}

	case ast.Identifier:
		// Treat identifiers as references (symbols to be resolved)
		name, err := nameWithModifiers(arg.Name, astInstruction.Modifier)
		if err != nil {
			return nil, err
		}
		ins.argument = reference{name: name}

	default:
		return nil, fmt.Errorf("unexpected argument type %T", arg)
	}

	return []ast.Node{ins}, nil
}

// modifierOffset computes the cumulative integer offset from a list of modifiers.
func modifierOffset(modifiers []ast.Modifier) (int64, error) {
	var offset int64
	for _, mod := range modifiers {
		v, err := number.Parse(mod.Value)
		if err != nil {
			return 0, fmt.Errorf("parsing modifier value '%s': %w", mod.Value, err)
		}
		switch mod.Operator.Operator {
		case "+":
			offset += int64(v)
		case "-":
			offset -= int64(v)
		default:
			return 0, fmt.Errorf("unsupported modifier operator '%s'", mod.Operator.Operator)
		}
	}
	return offset, nil
}

// nameWithModifiers appends the combined modifier offset to a symbol name in a format
// that parseReferenceOffset can parse (e.g. "noise+5" or "label-3").
func nameWithModifiers(name string, modifiers []ast.Modifier) (string, error) {
	if len(modifiers) == 0 {
		return name, nil
	}
	offset, err := modifierOffset(modifiers)
	if err != nil {
		return "", err
	}
	if offset >= 0 {
		return fmt.Sprintf("%s+%d", name, offset), nil
	}
	return fmt.Sprintf("%s%d", name, offset), nil // offset is negative, fmt includes '-'
}

func parseInclude[T any](asm *parseAST[T], inc ast.Include) ([]ast.Node, error) {
	if !inc.Binary {
		return nil, errors.New("non binary includes are currently not supported") // TODO implement
	}

	name := strings.Trim(inc.Name, "\"'")
	b, err := asm.fileReader(name)
	if err != nil {
		return nil, fmt.Errorf("reading file '%s': %w", name, err)
	}

	dat := &data{size: expression.New()}
	dat.size.SetValue(1)
	dat.values = append(dat.values, b)
	return []ast.Node{dat}, nil
}

func parseVariable(astVar ast.Variable) []ast.Node {
	v := &variable{v: astVar}
	return []ast.Node{v}
}

func parseFunction[T any](asm *parseAST[T], fun ast.Function) ([]ast.Node, error) {
	sym, err := scope.NewSymbol(asm.currentScope, fun.Name, scope.FunctionType)
	if err != nil {
		return nil, fmt.Errorf("creating symbol: %w", err)
	}

	asm.currentScope = scope.New(asm.currentScope)
	newScope := scopeChange{
		scope: asm.currentScope,
	}

	return []ast.Node{newScope, &symbol{Symbol: sym}}, nil
}

func parseFunctionEnd[T any](asm *parseAST[T], _ ast.FunctionEnd) ([]ast.Node, error) {
	parentScope := asm.currentScope.Parent()
	if parentScope == nil {
		return nil, errors.New("unexpected function end, no parent scope found")
	}

	asm.currentScope = parentScope

	newScope := scopeChange{
		scope: asm.currentScope,
	}

	return []ast.Node{newScope}, nil
}

func parseMacro(astMacro ast.Macro) ([]ast.Node, error) {
	mac := macro{
		name:      astMacro.Name,
		arguments: map[string]int{},
		tokens:    astMacro.Token,
	}

	for i, argument := range astMacro.Arguments {
		_, ok := mac.arguments[argument]
		if ok {
			return nil, fmt.Errorf("macro argument '%s' found twice", argument)
		}
		mac.arguments[argument] = i
	}

	return []ast.Node{mac}, nil
}
