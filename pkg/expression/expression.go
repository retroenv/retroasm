// Package expression implements an expression parser and evaluator for assembly language expressions.
//
// This package provides a complete expression evaluation system using the Shunting Yard algorithm
// for parsing infix expressions into Reverse Polish Notation (RPN) and evaluating them.
//
// Key features:
//   - Mathematical operations: +, -, *, /, %, ^ (exponentiation)
//   - Comparison operations: ==, <, <=, >, >=
//   - Parentheses for grouping and precedence control
//   - Symbol resolution from assembly scopes
//   - Program counter ($) references for address calculations
//   - Mixed data types: int64, []byte, bool
//   - Circular dependency detection
//   - Lazy evaluation with caching support
//
// The expression evaluator supports two evaluation modes:
//   - Immediate evaluation: Expression is evaluated when requested
//   - Deferred evaluation: Expression contains program counter ($) references
//     and must be evaluated during the address assignment phase
//
// Usage:
//
//	// Create expression from tokens
//	expr := expression.New(tokens...)
//
//	// Evaluate with scope context
//	result, err := expr.Evaluate(scope, dataWidth)
//
//	// Evaluate with program counter context
//	result, err := expr.EvaluateAtProgramCounter(scope, dataWidth, pc)
package expression

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/retroenv/retroasm/pkg/lexer/token"
	"github.com/retroenv/retroasm/pkg/number"
	"github.com/retroenv/retroasm/pkg/scope"
)

var (
	errCircularDependency      = errors.New("circular symbol dependency detected")
	errDivisionByZero          = errors.New("division by zero")
	errEvaluateAtAddressAssign = errors.New("expression can not be referenced due to program counter $ usage")
	errExpressionNotEvaluated  = errors.New("expression is not evaluated")
	errMismatchedParenthesis   = errors.New("mismatched parenthesis found")
)

// ProgramCounterReference references the current program address in an expression.
// This can be used in the size expression of data references to create padding.
var ProgramCounterReference = "$"

// Expression represents an expression or value.
type Expression struct {
	nodes []token.Token

	value      any  // contains the calculated value, can be of type int64, []byte or bool
	evaluated  bool // if evaluated and only once evaluating, the value can be returned
	evaluating bool // evaluation in progress flag to detect circular dependencies

	// = expressions are evaluated once, EQU on every usage
	evaluateOnce bool
	// if an expression uses $ to refer to the current program counter,
	// it can only be evaluated at the assembler address assigning step
	evaluateAtAddressAssign bool
}

// New creates a new expression and adds all the passed tokens.
func New(tokens ...token.Token) *Expression {
	e := &Expression{}
	e.AddTokens(tokens...)
	return e
}

// Copy creates a copy of the expression.
func (e *Expression) Copy() *Expression {
	return &Expression{
		nodes:                   slices.Clone(e.nodes),
		value:                   e.value,
		evaluated:               e.evaluated,
		evaluating:              e.evaluating,
		evaluateOnce:            e.evaluateOnce,
		evaluateAtAddressAssign: e.evaluateAtAddressAssign,
	}
}

// CopyExpression creates a copy of the expression.
// Secondary copy function to avoid cyclic dependency.
func (e *Expression) CopyExpression() any {
	return e.Copy()
}

// SetEvaluateOnce sets the evaluate once flag for the expression.
// = will declare an alias that is evaluated on processing of the node,
// EQU will declare an expression that is evaluated on every usage and
// has access to updated/overwritten values of = aliases.
func (e *Expression) SetEvaluateOnce(evaluateOnce bool) {
	e.evaluateOnce = evaluateOnce
}

// IsEvaluatedOnce returns whether the expression is only evaluated once.
func (e *Expression) IsEvaluatedOnce() bool {
	return e.evaluateOnce
}

// IsEvaluatedAtAddressAssign returns whether the expression is only evaluated
// at the assembler address assigning step.
func (e *Expression) IsEvaluatedAtAddressAssign() bool {
	return e.evaluateAtAddressAssign
}

// AddTokens adds tokens to the expression.
func (e *Expression) AddTokens(tokens ...token.Token) {
	for _, tok := range tokens {
		if tok.Type == token.Number && tok.Value == ProgramCounterReference {
			e.evaluateAtAddressAssign = true
		}
		e.nodes = append(e.nodes, tok)
	}
}

// Tokens returns the tokens of the expression.
func (e *Expression) Tokens() []token.Token {
	return e.nodes
}

// IntValue returns the int value of the expression, it will return an error
// if the expression is not evaluated or resulted in a different type than int64.
func (e *Expression) IntValue() (int64, error) {
	if !e.evaluated {
		return 0, errExpressionNotEvaluated
	}
	i, ok := e.value.(int64)
	if !ok {
		return 0, fmt.Errorf("unexpected expression value type %T", e.value)
	}
	return i, nil
}

// SetValue sets the evaluated value of the expression and marks it as evaluated.
// This is useful in case the stored value needs to be updated.
func (e *Expression) SetValue(value int64) {
	e.value = value
	e.evaluated = true
}

// Evaluate the expression. The returned value can be of can be of type int64, []byte or bool.
func (e *Expression) Evaluate(scope *scope.Scope, dataWidth int) (any, error) {
	if dataWidth < 0 {
		return 0, fmt.Errorf("invalid data width: %d", dataWidth)
	}
	if e.evaluated && e.evaluateOnce {
		return e.value, nil
	}
	if e.evaluating {
		return 0, errCircularDependency
	}
	if e.evaluateAtAddressAssign {
		return 0, errEvaluateAtAddressAssign
	}

	return e.evaluate(scope, dataWidth, 0)
}

// EvaluateAtProgramCounter evaluates the expression using the current program counter.
// The returned value can be of can be of type int64, []byte or bool.
func (e *Expression) EvaluateAtProgramCounter(scope *scope.Scope, dataWidth int, programCounter uint64) (any, error) {
	if dataWidth < 0 {
		return 0, fmt.Errorf("invalid data width: %d", dataWidth)
	}
	return e.evaluate(scope, dataWidth, programCounter)
}

func (e *Expression) evaluate(scope *scope.Scope, dataWidth int, programCounter uint64) (any, error) {
	e.evaluating = true

	rpn, err := parseToRPN(scope, e.nodes, programCounter)
	if err != nil {
		return 0, fmt.Errorf("parsing expression to RPN: %w", err)
	}

	e.value, err = evaluateRPN(rpn, dataWidth)
	if err != nil {
		return 0, fmt.Errorf("evaluating RPN nodes: %w", err)
	}

	e.evaluated = true
	e.evaluating = false
	return e.value, nil
}

// nolint: funlen, cyclop
func parseToRPN(scope *scope.Scope, nodes []token.Token, programCounter uint64) ([]token.Token, error) {
	values := &stack[token.Token]{}
	operators := &stack[token.Token]{}

	for i := 0; i < len(nodes); i++ {
		tok := nodes[i]

		switch tok.Type {
		case token.Identifier:
			symbolTokens, err := parseToRPNHandleIdentifier(scope, tok, values)
			if err != nil {
				return nil, err
			}

			if len(symbolTokens) > 0 {
				// nolint: gocritic
				items := append(nodes[:i], symbolTokens...)
				nodes = append(items, nodes[i+1:]...)
				i--
			}

		case token.Number:
			if tok.Value == ProgramCounterReference {
				tok.Value = strconv.FormatUint(programCounter, 10)
			}
			values.push(tok)

		case token.LeftParentheses:
			operators.push(tok)

		case token.RightParentheses:
			foundLeftParenthesis := false

			for operators.len() > 0 {
				op := operators.pop()
				if op.Type == token.LeftParentheses {
					foundLeftParenthesis = true
					break
				}
				values.push(op)
			}
			if !foundLeftParenthesis {
				return nil, fmt.Errorf("%w: missing left parenthesis", errMismatchedParenthesis)
			}

		default:
			if err := parseToRPNHandleOperator(tok, values, operators); err != nil {
				return nil, err
			}
		}
	}

	// process remaining operators
	for range operators.len() {
		operator := operators.pop()
		if operator.Type == token.LeftParentheses {
			return nil, fmt.Errorf("%w: missing right parenthesis", errMismatchedParenthesis)
		}
		values.push(operator)
	}

	return values.data, nil
}

func parseToRPNHandleIdentifier(scope *scope.Scope, tok token.Token, values *stack[token.Token]) ([]token.Token, error) {
	if tok.Value[0] == '"' || tok.Value[0] == '\'' {
		values.push(tok)
		return nil, nil
	}

	sym, err := scope.GetSymbol(tok.Value)
	if err != nil {
		return nil, fmt.Errorf("getting expression symbol '%s': %w", tok.Value, err)
	}

	var symbolTokens []token.Token

	value, err := sym.Value(scope)
	if err != nil {
		if errors.Is(err, errCircularDependency) {
			return nil, fmt.Errorf("getting symbol value: %w", err)
		}

		// if symbol can't be evaluated, replace the current token with the
		// tokens of the symbol
		exp := sym.Expression()
		symbolTokens = exp.Tokens()
		if len(symbolTokens) == 0 {
			return nil, fmt.Errorf("getting symbol value: %w", err)
		}
		return symbolTokens, nil
	}

	switch v := value.(type) {
	case uint64:
		tok.Value = strconv.FormatUint(v, 10)
	case int64:
		tok.Value = strconv.FormatInt(v, 10)
	case []byte:
		tok.Value = string(v)
	default:
		return nil, fmt.Errorf("unsupported expression value type %T", value)
	}

	values.push(tok)
	return nil, nil
}

func parseToRPNHandleOperator(tok token.Token, values, operators *stack[token.Token]) error {
	priorityInfo, ok := operatorPriority[tok.Type]
	if !ok {
		return fmt.Errorf("unexpected operator token: %d", tok.Type)
	}
	rightAssociative := priorityInfo.rightAssociative

	for operators.len() > 0 {
		top := operators.last()
		if top.Type == token.LeftParentheses {
			break
		}

		previousPriorityInfo := operatorPriority[top.Type]

		if (rightAssociative && priorityInfo.priority < previousPriorityInfo.priority) ||
			(!rightAssociative && priorityInfo.priority <= previousPriorityInfo.priority) {

			operators.pop()
			values.push(top)
		} else {
			break
		}
	}

	operators.push(tok)
	return nil
}

// evaluateRPN evaluates a list of RPNTokens and returns the calculated value.
func evaluateRPN(tokens []token.Token, dataWidth int) (any, error) {
	if tokens == nil {
		return int64(0), nil
	}

	hasOperators := false
	values := &stack[any]{
		data: make([]any, 0, len(tokens)),
	}

	for _, tok := range tokens {
		if !tok.Type.IsOperator() {
			// push all operands to the stack
			if tok.Value[0] == '"' || tok.Value[0] == '\'' {
				s := strings.Trim(tok.Value, "\"'")
				b := []byte(s)
				values.push(b)
			} else {
				i, err := number.Parse(tok.Value)
				if err != nil {
					return 0, fmt.Errorf("parsing number '%s': %w", tok.Value, err)
				}
				values.push(int64(i))
			}
			continue
		}

		hasOperators = true

		// execute current operator
		if values.len() < 2 {
			return 0, fmt.Errorf("missing operand, expected 2 but found %d", values.len())
		}

		arg2 := values.pop()
		arg1 := values.pop()

		val, err := evaluateOperator(tok.Type, arg1, arg2)
		if err != nil {
			return 0, err
		}

		// push result back to stack
		values.push(val)
	}

	if values.len() != 1 {
		if hasOperators {
			return 0, fmt.Errorf("stack corrupted, expected 1 item but found %d", values.len())
		}
		data, err := processData(tokens, dataWidth)
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	result := values.last()
	return result, nil
}

func processData(tokens []token.Token, dataWidth int) ([]byte, error) {
	data := make([]byte, 0, len(tokens)*dataWidth)

	for _, tok := range tokens {
		switch tok.Type {
		case token.Identifier:
			//  unescape string
			s := strings.Trim(tok.Value, "\"'")
			data = append(data, []byte(s)...)

		case token.Number:
			i, err := number.Parse(tok.Value)
			if err != nil {
				return nil, fmt.Errorf("parsing number '%s': %w", tok.Value, err)
			}
			if err := number.CheckDataWidth(i, dataWidth); err != nil {
				return nil, fmt.Errorf("checking data byte width: %w", err)
			}
			b, err := number.WriteToBytes(i, dataWidth)
			if err != nil {
				return nil, fmt.Errorf("writing number as bytes: %w", err)
			}
			data = append(data, b...)

		default:
			return nil, fmt.Errorf("unsupported value type %T", tok.Type)
		}
	}

	return data, nil
}
