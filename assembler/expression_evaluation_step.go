package assembler

import (
	"errors"
	"fmt"

	"github.com/retroenv/assembler/number"
	"github.com/retroenv/assembler/parser/ast"
	"github.com/retroenv/assembler/scope"
)

var (
	errExpressionCantReferenceProgramCounter = errors.New("expression can not reference program counter")
	errConditionOutsideIfContext             = errors.New("directive used outside if context")
)

// evaluateExpressionsStep parses the AST nodes and evaluates aliases to their values.
func evaluateExpressionsStep(asm *Assembler) error {
	asm.cfg.Reset()

	for _, seg := range asm.segmentsOrder {
		nodes := make([]any, 0, len(seg.nodes))

		for _, node := range seg.nodes {
			remove, err := evaluateNode(asm, node)
			if err != nil {
				return err
			}
			if !remove {
				nodes = append(nodes, node)
			}
		}

		seg.nodes = nodes
	}

	if asm.currentContext.parent != nil {
		return errors.New("missing endif")
	}
	return nil
}

// evaluateNode evaluates a node and returns whether the node should be removed.
// This is useful for conditional nodes with an expression that does not match and
// that wraps other nodes.
func evaluateNode(asm *Assembler, node any) (bool, error) {
	// always handle conditional nodes
	switch n := node.(type) {
	case *ast.If:
		return true, parseIfCondition(asm, n)
	case *ast.Ifdef:
		parseIfdefCondition(asm, n)
		return true, nil
	case *ast.Ifndef:
		parseIfndefCondition(asm, n)
		return true, nil
	case *ast.Else:
		return true, processElseCondition(asm)
	case *ast.ElseIf:
		return true, parseElseIfCondition(asm, n)
	case *ast.Endif:
		return true, processEndifCondition(asm)
	}

	// skip processing nodes if the if context condition is not met
	if !asm.currentContext.processNodes {
		return true, nil
	}

	switch n := node.(type) {
	case *data:
		return false, parseDataExpression(asm, n)

	case *base:
		_, err := n.address.Evaluate(asm.currentScope, asm.cfg.Arch.AddressWidth)
		if err != nil {
			return false, fmt.Errorf("evaluating base expression: %w", err)
		}

	case *scope.Symbol:
		return false, parseSymbolExpression(asm, n)

	case *ast.Configuration:
		if err := parseConfigExpression(asm, n); err != nil {
			return false, err
		}

		if n.Item == ast.ConfigFillValue {
			asm.cfg.FillValues = n.Expression
		}
	}

	return false, nil
}

func parseDataExpression(asm *Assembler, dat *data) error {
	if !dat.size.IsEvaluatedAtAddressAssign() {
		_, err := dat.size.Evaluate(asm.currentScope, dat.width)
		if err != nil {
			return fmt.Errorf("evaluating data size expression: %w", err)
		}
	}

	// if no fill value expression is specified, use the current fill value config expression
	if dat.expression == nil {
		dat.expression = asm.cfg.FillValues
	}
	if dat.expression == nil || dat.expression.IsEvaluatedAtAddressAssign() {
		return nil
	}

	value, err := dat.expression.Evaluate(asm.currentScope, dat.width)
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

func parseSymbolExpression(asm *Assembler, sym *scope.Symbol) error {
	exp := sym.Expression()
	if exp == nil || exp.IsEvaluatedAtAddressAssign() {
		return nil
	}

	// only process constant expressions that result in a value
	if exp.IsEvaluatedOnce() {
		_, err := exp.Evaluate(asm.currentScope, 1)
		if err != nil {
			return fmt.Errorf("evaluating symbol expression: %w", err)
		}
	}

	if sym.Type() == scope.AliasType {
		if err := asm.currentScope.AddSymbol(sym); err != nil {
			return fmt.Errorf("setting symbol in scope: %w", err)
		}
	}

	return nil
}

func parseConfigExpression(asm *Assembler, cfg *ast.Configuration) error {
	exp := cfg.Expression
	if exp == nil {
		return nil
	}

	if exp.IsEvaluatedAtAddressAssign() {
		return errExpressionCantReferenceProgramCounter
	}

	// only process constant expressions that result in a value
	if exp.IsEvaluatedOnce() {
		_, err := exp.Evaluate(asm.currentScope, 1)
		if err != nil {
			return fmt.Errorf("evaluating config expression: %w", err)
		}
	}

	return nil
}

func parseIfCondition(asm *Assembler, cond *ast.If) error {
	if cond.Condition.IsEvaluatedAtAddressAssign() {
		return errExpressionCantReferenceProgramCounter
	}

	value, err := cond.Condition.Evaluate(asm.currentScope, asm.cfg.Arch.AddressWidth)
	if err != nil {
		return fmt.Errorf("evaluating if condition at program counter: %w", err)
	}

	conditionMet, ok := value.(bool)
	if !ok {
		return fmt.Errorf("unsupported expression value type %T", value)
	}

	ctx := &context{
		processNodes: conditionMet,
		parent:       asm.currentContext,
	}
	asm.currentContext = ctx
	return nil
}

func parseIfdefCondition(asm *Assembler, cond *ast.Ifdef) {
	conditionMet := true
	_, err := asm.currentScope.GetSymbol(cond.Identifier)
	if err != nil {
		conditionMet = false
	}

	ctx := &context{
		processNodes: conditionMet,
		parent:       asm.currentContext,
	}
	asm.currentContext = ctx
}

func parseIfndefCondition(asm *Assembler, cond *ast.Ifndef) {
	conditionMet := false
	_, err := asm.currentScope.GetSymbol(cond.Identifier)
	if err != nil {
		conditionMet = true
	}

	ctx := &context{
		processNodes: conditionMet,
		parent:       asm.currentContext,
	}
	asm.currentContext = ctx
}

func parseElseIfCondition(asm *Assembler, cond *ast.ElseIf) error {
	if asm.currentContext.parent == nil {
		return errConditionOutsideIfContext
	}
	if cond.Condition.IsEvaluatedAtAddressAssign() {
		return errExpressionCantReferenceProgramCounter
	}

	value, err := cond.Condition.Evaluate(asm.currentScope, asm.cfg.Arch.AddressWidth)
	if err != nil {
		return fmt.Errorf("evaluating if condition at program counter: %w", err)
	}

	conditionMet, ok := value.(bool)
	if !ok {
		return fmt.Errorf("unsupported expression value type %T", value)
	}

	asm.currentContext.processNodes = conditionMet
	return nil
}
