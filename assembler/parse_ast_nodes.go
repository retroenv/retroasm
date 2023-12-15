package assembler

import (
	"errors"
	"fmt"
	"strings"

	"github.com/retroenv/assembler/expression"
	"github.com/retroenv/assembler/lexer/token"
	"github.com/retroenv/assembler/number"
	"github.com/retroenv/assembler/parser/ast"
	"github.com/retroenv/assembler/scope"
)

// parseASTNodesStep parses the AST nodes and converts them to internal types.
func parseASTNodesStep(asm *Assembler) error {
	nodes, err := asm.parser.Read()
	if err != nil {
		return fmt.Errorf("parsing lexer tokens: %w", err)
	}

	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.Comment:

		case *ast.Segment:
			if err := parseSegment(asm, n); err != nil {
				return fmt.Errorf("parsing segment node: %w", err)
			}

		default:
			if asm.currentSegment == nil {
				return errNoCurrentSegment
			}

			if err := parseASTNode(asm, node); err != nil {
				return err
			}
		}
	}

	return nil
}

// nolint: cyclop, funlen
func parseASTNode(asm *Assembler, node ast.Node) error {
	switch n := node.(type) {
	case *ast.Data:
		if err := parseData(asm, n); err != nil {
			return fmt.Errorf("parsing data node: %w", err)
		}

	case *ast.Alias:
		if err := parseAlias(asm, n); err != nil {
			return fmt.Errorf("parsing alias node: %w", err)
		}

	case *ast.Label:
		if err := parseLabel(asm, n); err != nil {
			return fmt.Errorf("parsing label node: %w", err)
		}

	case *ast.Function:
		if err := parseFunction(asm, n); err != nil {
			return fmt.Errorf("parsing function node: %w", err)
		}

	case ast.FunctionEnd:
		if err := parseFunctionEnd(asm, n); err != nil {
			return fmt.Errorf("parsing function end node: %w", err)
		}

	case *ast.Instruction:
		if err := parseInstruction(asm, n); err != nil {
			return fmt.Errorf("parsing instruction node: %w", err)
		}

	case *ast.Include:
		if err := parseInclude(asm, n); err != nil {
			return fmt.Errorf("parsing include node: %w", err)
		}

	case *ast.Base:
		parseBase(asm, n)

	case *ast.Variable:
		parseVariable(asm, n)

	case *ast.Configuration,
		*ast.If,
		*ast.Ifdef,
		*ast.Ifndef,
		*ast.Else,
		*ast.ElseIf,
		*ast.Endif:

		asm.currentSegment.addNode(n)

	default:
		return fmt.Errorf("unsupported node type %T", n)
	}

	return nil
}

func parseSegment(asm *Assembler, astSegment *ast.Segment) error {
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

func parseData(asm *Assembler, astData *ast.Data) error {
	dat := &data{
		fill:  astData.Fill,
		width: astData.Width,
		size:  astData.Size,
	}
	if dat.size == nil {
		dat.size = expression.New()
	}

	switch astData.Type {
	case "address":
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
			return fmt.Errorf("parsing data address: %w", err)
		}

	case "data":
		dat.expression = astData.Values

	default:
		return fmt.Errorf("unsupported data type '%s'", astData.Type)
	}

	asm.currentSegment.addNode(dat)
	return nil
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

func parseAlias(asm *Assembler, alias *ast.Alias) error {
	typ := scope.AliasType
	if !alias.SymbolReusable {
		typ = scope.EquType
	}

	sym, err := scope.NewSymbol(asm.currentScope, alias.Name, typ)
	if err != nil {
		return fmt.Errorf("creating symbol: %w", err)
	}
	sym.SetExpression(alias.Expression)
	asm.currentSegment.addNode(sym)
	return nil
}

func parseLabel(asm *Assembler, label *ast.Label) error {
	sym, err := scope.NewSymbol(asm.currentScope, label.Name, scope.LabelType)
	if err != nil {
		return fmt.Errorf("creating symbol: %w", err)
	}

	asm.currentSegment.addNode(sym)
	return nil
}

func parseInstruction(asm *Assembler, astInstruction *ast.Instruction) error {
	if astInstruction.Modifier != nil {
		return fmt.Errorf("unexpected modifier %v", astInstruction.Modifier)
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

	case *ast.Label:
		ins.argument = reference{name: arg.Name}

	default:
		return fmt.Errorf("unexpected argument type %T", arg)
	}

	asm.currentSegment.addNode(ins)
	return nil
}

func parseInclude(asm *Assembler, inc *ast.Include) error {
	if !inc.Binary {
		return errors.New("non binary includes are currently not supported") // TODO implement
	}

	name := strings.Trim(inc.Name, "\"'")
	b, err := asm.fileReader(name)
	if err != nil {
		return fmt.Errorf("reading file '%s': %w", name, err)
	}

	dat := &data{size: expression.New()}
	dat.size.SetValue(1)
	dat.values = append(dat.values, b)
	asm.currentSegment.addNode(dat)
	return nil
}

func parseBase(asm *Assembler, astBase *ast.Base) {
	bas := &base{address: astBase.Address}
	asm.currentSegment.addNode(bas)
}

func parseVariable(asm *Assembler, astVar *ast.Variable) {
	v := &variable{v: astVar}
	asm.currentSegment.addNode(v)
}

func parseFunction(asm *Assembler, fun *ast.Function) error {
	sym, err := scope.NewSymbol(asm.currentScope, fun.Name, scope.FunctionType)
	if err != nil {
		return fmt.Errorf("creating symbol: %w", err)
	}

	asm.currentScope = scope.New(asm.currentScope)
	newScope := scopeChange{
		scope: asm.currentScope,
	}
	asm.currentSegment.addNode(newScope)

	asm.currentSegment.addNode(sym)

	return nil
}

func parseFunctionEnd(asm *Assembler, _ ast.FunctionEnd) error {
	parentScope := asm.currentScope.Parent()
	if parentScope == nil {
		return errors.New("unexpected function end, no parent scope found")
	}

	asm.currentScope = parentScope

	newScope := scopeChange{
		scope: asm.currentScope,
	}
	asm.currentSegment.addNode(newScope)

	return nil
}
