package assembler

import (
	"errors"
	"fmt"

	"github.com/retroenv/assembler/arch"
	"github.com/retroenv/assembler/expression"
	"github.com/retroenv/assembler/number"
	"github.com/retroenv/assembler/parser/ast"
	"github.com/retroenv/assembler/scope"
)

var (
	errExpressionCantReferenceProgramCounter = errors.New("expression can not reference program counter")
	errConditionOutsideIfContext             = errors.New("directive used outside if context")
)

type expressionEvaluation struct {
	arch arch.Architecture

	currentContext *context
	currentScope   *scope.Scope // current scope, can be a function scope with file scope as parent

	fillValues *expression.Expression
}

// evaluateExpressionsStep parses the AST nodes and evaluates aliases to their values.
func evaluateExpressionsStep(asm *Assembler) error {
	expEval := expressionEvaluation{
		arch:         asm.cfg.Arch,
		currentScope: asm.fileScope,
		currentContext: &context{
			processNodes: true,
			parent:       nil,
		},
	}

	for _, seg := range asm.segmentsOrder {
		nodes := make([]any, 0, len(seg.nodes))

		for _, node := range seg.nodes {
			remove, err := evaluateNode(&expEval, node)
			if err != nil {
				return err
			}
			if !remove {
				nodes = append(nodes, node)
			}
		}

		seg.nodes = nodes
	}

	if expEval.currentContext.parent != nil {
		return errors.New("missing endif")
	}
	return nil
}

// evaluateNode evaluates a node and returns whether the node should be removed.
// This is useful for conditional nodes with an expression that does not match and
// that wraps other nodes.
// nolint:cyclop
func evaluateNode(expEval *expressionEvaluation, node any) (bool, error) {
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
		_, err := n.Address.Evaluate(expEval.currentScope, expEval.arch.AddressWidth)
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
		_, err := n.Address.Evaluate(expEval.currentScope, expEval.arch.AddressWidth)
		if err != nil {
			return false, fmt.Errorf("evaluating enum expression: %w", err)
		}

	case *data:
		return false, parseDataExpression(expEval, n)

	case *scope.Symbol:
		return false, parseSymbolExpression(expEval, n)
	}

	return false, nil
}

func parseDataExpression(expEval *expressionEvaluation, dat *data) error {
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

func parseSymbolExpression(expEval *expressionEvaluation, sym *scope.Symbol) error {
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
		if err := expEval.currentScope.AddSymbol(sym); err != nil {
			return fmt.Errorf("setting symbol in scope: %w", err)
		}
	}

	return nil
}

func parseConfigExpression(expEval *expressionEvaluation, cfg ast.Configuration) error {
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

func parseIfCondition(expEval *expressionEvaluation, cond ast.If) error {
	if cond.Condition.IsEvaluatedAtAddressAssign() {
		return errExpressionCantReferenceProgramCounter
	}

	value, err := cond.Condition.Evaluate(expEval.currentScope, expEval.arch.AddressWidth)
	if err != nil {
		return fmt.Errorf("evaluating if condition at program counter: %w", err)
	}

	conditionMet, ok := value.(bool)
	if !ok {
		return fmt.Errorf("unsupported expression value type %T", value)
	}

	ctx := &context{
		processNodes: conditionMet,
		parent:       expEval.currentContext,
	}
	expEval.currentContext = ctx
	return nil
}

func parseIfdefCondition(expEval *expressionEvaluation, cond ast.Ifdef) {
	conditionMet := true
	_, err := expEval.currentScope.GetSymbol(cond.Identifier)
	if err != nil {
		conditionMet = false
	}

	ctx := &context{
		processNodes: conditionMet,
		parent:       expEval.currentContext,
	}
	expEval.currentContext = ctx
}

func parseIfndefCondition(expEval *expressionEvaluation, cond ast.Ifndef) {
	conditionMet := false
	_, err := expEval.currentScope.GetSymbol(cond.Identifier)
	if err != nil {
		conditionMet = true
	}

	ctx := &context{
		processNodes: conditionMet,
		parent:       expEval.currentContext,
	}
	expEval.currentContext = ctx
}

func parseElseIfCondition(expEval *expressionEvaluation, cond ast.ElseIf) error {
	if expEval.currentContext.parent == nil {
		return errConditionOutsideIfContext
	}
	if cond.Condition.IsEvaluatedAtAddressAssign() {
		return errExpressionCantReferenceProgramCounter
	}

	value, err := cond.Condition.Evaluate(expEval.currentScope, expEval.arch.AddressWidth)
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
