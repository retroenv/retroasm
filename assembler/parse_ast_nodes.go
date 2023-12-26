package assembler

import (
	"errors"
	"fmt"
	"strings"

	"github.com/retroenv/assembler/assembler/config"
	"github.com/retroenv/assembler/expression"
	"github.com/retroenv/assembler/lexer/token"
	"github.com/retroenv/assembler/number"
	"github.com/retroenv/assembler/parser"
	"github.com/retroenv/assembler/parser/ast"
	"github.com/retroenv/assembler/scope"
)

type parseAST struct {
	cfg *config.Config
	// a function that reads in a file, for testing includes, defaults to os.ReadFile
	fileReader func(name string) ([]byte, error)

	currentScope   *scope.Scope // current scope, can be a function scope with file scope as parent
	currentSegment *segment     // the current segment being parsed

	segments      map[string]*segment // maps segment name to segment
	segmentsOrder []*segment          // sorted list of all parsed segments
}

// parseASTNodesStep parses the AST nodes and converts them to internal types.
func parseASTNodesStep(asm *Assembler) error {
	p := &parseAST{
		cfg:          asm.cfg,
		fileReader:   asm.fileReader,
		currentScope: asm.fileScope,
		segments:     map[string]*segment{},
	}

	pars := parser.New(asm.cfg.Arch, asm.inputReader)
	if err := pars.Read(); err != nil {
		return fmt.Errorf("parsing lexer tokens: %w", err)
	}
	nodes, err := pars.TokensToAstNodes()
	if err != nil {
		return fmt.Errorf("converting tokens to ast nodes: %w", err)
	}

	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.Comment:

		case ast.Segment:
			if err := parseSegment(p, n); err != nil {
				return fmt.Errorf("parsing segment node: %w", err)
			}

		default:
			if p.currentSegment == nil {
				return errNoCurrentSegment
			}

			newNodes, err := parseASTNode(p, node)
			if err != nil {
				return err
			}
			for _, newNode := range newNodes {
				p.currentSegment.addNode(newNode)
			}
		}
	}

	asm.segments = p.segments
	asm.segmentsOrder = p.segmentsOrder

	return nil
}

// nolint: cyclop, funlen
func parseASTNode(asm *parseAST, node ast.Node) ([]any, error) {
	var (
		err   error
		nodes []any
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
	case ast.Configuration,
		ast.If,
		ast.Ifdef,
		ast.Ifndef,
		ast.Else,
		ast.ElseIf,
		ast.Endif,
		ast.Identifier,
		ast.Error,
		ast.Base:

		return []any{n}, nil

	default:
		return nil, fmt.Errorf("unsupported node type %T", n)
	}

	if err != nil {
		return nil, fmt.Errorf("parsing node type %T: %w", node, err)
	}
	return nodes, nil
}

func parseSegment(asm *parseAST, astSegment ast.Segment) error {
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

var errNoCurrentSegment = errors.New("no current segment found")

func parseData(astData ast.Data) ([]any, error) {
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

	return []any{dat}, nil
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

func parseAlias(asm *parseAST, alias ast.Alias) ([]any, error) {
	typ := scope.AliasType
	if !alias.SymbolReusable {
		typ = scope.EquType
	}

	sym, err := scope.NewSymbol(asm.currentScope, alias.Name, typ)
	if err != nil {
		return nil, fmt.Errorf("creating symbol: %w", err)
	}
	sym.SetExpression(alias.Expression)
	return []any{sym}, nil
}

func parseLabel(asm *parseAST, label ast.Label) ([]any, error) {
	sym, err := scope.NewSymbol(asm.currentScope, label.Name, scope.LabelType)
	if err != nil {
		return nil, fmt.Errorf("creating symbol: %w", err)
	}

	return []any{sym}, nil
}

func parseInstruction(astInstruction ast.Instruction) ([]any, error) {
	if astInstruction.Modifier != nil {
		return nil, fmt.Errorf("unexpected modifier %v", astInstruction.Modifier)
	}

	ins := &instruction{
		name:       astInstruction.Name,
		addressing: astInstruction.Addressing,
		argument:   astInstruction.Argument,
	}

	switch arg := astInstruction.Argument.(type) {
	case nil:

	case ast.Number:
		ins.argument = arg.Value

	case ast.Label:
		ins.argument = reference{name: arg.Name}

	default:
		return nil, fmt.Errorf("unexpected argument type %T", arg)
	}

	return []any{ins}, nil
}

func parseInclude(asm *parseAST, inc ast.Include) ([]any, error) {
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
	return []any{dat}, nil
}

func parseVariable(astVar ast.Variable) []any {
	v := &variable{v: astVar}
	return []any{v}
}

func parseFunction(asm *parseAST, fun ast.Function) ([]any, error) {
	sym, err := scope.NewSymbol(asm.currentScope, fun.Name, scope.FunctionType)
	if err != nil {
		return nil, fmt.Errorf("creating symbol: %w", err)
	}

	asm.currentScope = scope.New(asm.currentScope)
	newScope := scopeChange{
		scope: asm.currentScope,
	}

	return []any{newScope, sym}, nil
}

func parseFunctionEnd(asm *parseAST, _ ast.FunctionEnd) ([]any, error) {
	parentScope := asm.currentScope.Parent()
	if parentScope == nil {
		return nil, errors.New("unexpected function end, no parent scope found")
	}

	asm.currentScope = parentScope

	newScope := scopeChange{
		scope: asm.currentScope,
	}

	return []any{newScope}, nil
}

func parseMacro(astMacro ast.Macro) ([]any, error) {
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

	return []any{mac}, nil
}
