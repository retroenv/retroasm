package assembler

import (
	"errors"
	"fmt"

	"github.com/retroenv/retroasm/arch"
	"github.com/retroenv/retroasm/expression"
	"github.com/retroenv/retroasm/number"
	"github.com/retroenv/retroasm/parser/ast"
	"github.com/retroenv/retroasm/scope"
)

var (
	errExpressionCantReferenceProgramCounter = errors.New("expression can not reference program counter")
	errConditionOutsideIfContext             = errors.New("directive used outside of if context")
	errMissingEndif                          = errors.New("missing endif")
	errMultipleElseFound                     = errors.New("multiple else found")
	errReptCountNegative                     = errors.New("rept count can not be negative")
)

type expressionEvaluation[T any] struct {
	arch arch.Architecture[T]

	currentContext *conditionalContext
	currentScope   *scope.Scope // current scope, can be a function scope with file scope as parent

	fillValues *expression.Expression
}

// evaluateExpressionsStep parses the AST nodes and evaluates aliases to their values.
func evaluateExpressionsStep[T any](asm *Assembler[T]) error {
	expEval := expressionEvaluation[T]{
		arch:         asm.cfg.Arch,
		currentScope: asm.fileScope,
		currentContext: &conditionalContext{
			processNodes: true,
			parent:       nil,
		},
	}

	for segNr, seg := range asm.segmentsOrder {
		nodes := make([]ast.Node, 0, len(seg.nodes))

		// nolint:intrange // seg.nodes gets modified in the loop
		for nodeNr := 0; nodeNr < len(seg.nodes); nodeNr++ {
			node := seg.nodes[nodeNr]
			removeNode, err := evaluateNode[T](&expEval, seg, nodeNr, node)
			if err != nil {
				return fmt.Errorf("evaluating node %d in segment %d: %w", nodeNr, segNr, err)
			}
			if !removeNode {
				nodes = append(nodes, node)
			}
		}

		seg.nodes = nodes
	}

	if expEval.currentContext.parent != nil {
		return errMissingEndif
	}
	return nil
}

// evaluateNode evaluates a node and returns whether the node should be removed.
// This is useful for conditional nodes with an expression that does not match and
// that wraps other nodes.
// nolint:cyclop,funlen
func evaluateNode[T any](expEval *expressionEvaluation[T], seg *segment, currentNodeIndex int, node any) (bool, error) {
	// always handle conditional nodes
	switch n := node.(type) {
	case ast.If:
		return true, parseIfCondition(expEval, n)
	case ast.Ifdef:
		parseIfdefCondition(expEval, n)
		return true, nil
	case ast.Ifndef:
		parseIfndefCondition(expEval, n)
		return true, nil
	case ast.Else:
		return true, processElseCondition(expEval)
	case ast.ElseIf:
		return true, parseElseIfCondition(expEval, n)
	case ast.Endif:
		return true, processEndifCondition(expEval)
	case ast.Error:
		if expEval.currentContext.processNodes {
			return true, errors.New(n.Message)
		}
	}

	// skip processing nodes in case the if context condition is not met
	if !expEval.currentContext.processNodes {
		return true, nil
	}

	switch n := node.(type) {
	case ast.Base:
		_, err := n.Address.Evaluate(expEval.currentScope, expEval.arch.AddressWidth())
		if err != nil {
			return false, fmt.Errorf("evaluating base expression: %w", err)
		}

	case ast.Configuration:
		if err := parseConfigExpression(expEval, n); err != nil {
			return false, err
		}

		if n.Item == ast.ConfigFillValue {
			expEval.fillValues = n.Expression
		}

	case ast.Enum:
		_, err := n.Address.Evaluate(expEval.currentScope, expEval.arch.AddressWidth())
		if err != nil {
			return false, fmt.Errorf("evaluating enum expression: %w", err)
		}

	case ast.Rept:
		if err := parseRept(expEval, n, seg, currentNodeIndex); err != nil {
			return false, err
		}
		return true, nil

	case ast.Endr:
		return true, nil

	case *data:
		return false, parseDataExpression(expEval, n)

	case *symbol:
		return false, parseSymbolExpression(expEval, n)
	}

	return false, nil
}

func parseDataExpression[T any](expEval *expressionEvaluation[T], dat *data) error {
	if !dat.size.IsEvaluatedAtAddressAssign() {
		_, err := dat.size.Evaluate(expEval.currentScope, dat.width)
		if err != nil {
			return fmt.Errorf("evaluating data size expression: %w", err)
		}
	}

	// if no fill value expression is specified, use the current fill value config expression
	if dat.expression == nil {
		dat.expression = expEval.fillValues
	}
	if dat.expression == nil || dat.expression.IsEvaluatedAtAddressAssign() {
		return nil
	}

	value, err := dat.expression.Evaluate(expEval.currentScope, dat.width)
	if err != nil {
		return fmt.Errorf("evaluating data expression: %w", err)
	}

	switch v := value.(type) {
	case int64:
		b, err := number.WriteToBytes(uint64(v), dat.width)
		if err != nil {
			return fmt.Errorf("writing number as bytes: %w", err)
		}

		dat.values = append(dat.values, b)
		return nil

	case []byte:
		dat.values = append(dat.values, v)
		return nil

	default:
		return fmt.Errorf("unsupported expression value type %T", value)
	}
}

func parseSymbolExpression[T any](expEval *expressionEvaluation[T], sym *symbol) error {
	exp := sym.Expression()
	if exp == nil || exp.IsEvaluatedAtAddressAssign() {
		return nil
	}

	// only process constant expressions that result in a value
	if exp.IsEvaluatedOnce() {
		_, err := exp.Evaluate(expEval.currentScope, 1)
		if err != nil {
			return fmt.Errorf("evaluating symbol expression: %w", err)
		}
	}

	if sym.Type() == scope.AliasType {
		if err := expEval.currentScope.AddSymbol(sym.Symbol); err != nil {
			return fmt.Errorf("setting symbol in scope: %w", err)
		}
	}

	return nil
}

func parseConfigExpression[T any](expEval *expressionEvaluation[T], cfg ast.Configuration) error {
	exp := cfg.Expression
	if exp == nil {
		return nil
	}

	if exp.IsEvaluatedAtAddressAssign() {
		return errExpressionCantReferenceProgramCounter
	}

	// only process constant expressions that result in a value
	if exp.IsEvaluatedOnce() {
		_, err := exp.Evaluate(expEval.currentScope, 1)
		if err != nil {
			return fmt.Errorf("evaluating config expression: %w", err)
		}
	}

	return nil
}

func parseIfCondition[T any](expEval *expressionEvaluation[T], cond ast.If) error {
	if cond.Condition.IsEvaluatedAtAddressAssign() {
		return errExpressionCantReferenceProgramCounter
	}

	value, err := cond.Condition.Evaluate(expEval.currentScope, expEval.arch.AddressWidth())
	if err != nil {
		return fmt.Errorf("evaluating if condition at program counter: %w", err)
	}

	conditionMet, ok := value.(bool)
	if !ok {
		return fmt.Errorf("unsupported expression value type %T", value)
	}

	ctx := &conditionalContext{
		processNodes: conditionMet,
		parent:       expEval.currentContext,
	}
	expEval.currentContext = ctx
	return nil
}

func parseIfdefCondition[T any](expEval *expressionEvaluation[T], cond ast.Ifdef) {
	conditionMet := true
	_, err := expEval.currentScope.GetSymbol(cond.Identifier)
	if err != nil {
		conditionMet = false
	}

	ctx := &conditionalContext{
		processNodes: conditionMet,
		parent:       expEval.currentContext,
	}
	expEval.currentContext = ctx
}

func parseIfndefCondition[T any](expEval *expressionEvaluation[T], cond ast.Ifndef) {
	conditionMet := false
	_, err := expEval.currentScope.GetSymbol(cond.Identifier)
	if err != nil {
		conditionMet = true
	}

	ctx := &conditionalContext{
		processNodes: conditionMet,
		parent:       expEval.currentContext,
	}
	expEval.currentContext = ctx
}

func parseElseIfCondition[T any](expEval *expressionEvaluation[T], cond ast.ElseIf) error {
	if expEval.currentContext.parent == nil {
		return errConditionOutsideIfContext
	}
	if cond.Condition.IsEvaluatedAtAddressAssign() {
		return errExpressionCantReferenceProgramCounter
	}

	value, err := cond.Condition.Evaluate(expEval.currentScope, expEval.arch.AddressWidth())
	if err != nil {
		return fmt.Errorf("evaluating if condition at program counter: %w", err)
	}

	conditionMet, ok := value.(bool)
	if !ok {
		return fmt.Errorf("unsupported expression value type %T", value)
	}

	expEval.currentContext.processNodes = conditionMet
	return nil
}

func parseRept[T any](expEval *expressionEvaluation[T], rept ast.Rept, seg *segment, currentNodeIndex int) error {
	if rept.Count.IsEvaluatedAtAddressAssign() {
		return errExpressionCantReferenceProgramCounter
	}

	_, err := rept.Count.Evaluate(expEval.currentScope, expEval.arch.AddressWidth())
	if err != nil {
		return fmt.Errorf("evaluating if condition at program counter: %w", err)
	}

	count, err := rept.Count.IntValue()
	if err != nil {
		return fmt.Errorf("getting rept count: %w", err)
	}
	if count < 0 {
		return errReptCountNegative
	}

	var nodes []ast.Node
	var reptEnded bool

	for i := currentNodeIndex + 1; i < len(seg.nodes); i++ {
		node := seg.nodes[i]
		if _, ok := node.(ast.Endr); !ok {
			nodes = append(nodes, node)
		} else {
			reptEnded = true
			break
		}
	}

	if !reptEnded {
		return errors.New("rept without endr found")
	}

	unrollReptNodes(nodes, seg, currentNodeIndex, count)

	return nil
}

func unrollReptNodes(nodes []ast.Node, seg *segment, currentNodeIndex int, count int64) {
	nodesToInsert := make([]ast.Node, 0, len(nodes)*int(count))

	if count > 0 {
		// insert the nodes count-1 times, as the first insertion are the existing nodes
		count--
		for range count {
			for _, node := range nodes {
				nodesToInsert = append(nodesToInsert, node.Copy())
			}
		}

		// copy nodes up to endr
		nodes = seg.nodes[:currentNodeIndex+len(nodesToInsert)-1]
		// append now node copies
		nodes = append(nodes, nodesToInsert...)
		// append nodes after endr
		nodes = append(nodes, seg.nodes[currentNodeIndex+len(nodesToInsert):]...)
	} else {
		reptNodeCount := len(nodes)
		// copy nodes up to rept
		nodes = seg.nodes[:currentNodeIndex+1]
		// append nodes after endr
		nodes = append(nodes, seg.nodes[currentNodeIndex+reptNodeCount+2:]...)
	}

	seg.nodes = nodes
}
