package ast

import (
	"fmt"
	"reflect"
	"strings"
)

// InstructionArgumentCopier defines deep-copy behavior for an opaque typed
// instruction argument. Mutable architecture-specific values must implement it.
type InstructionArgumentCopier interface {
	CopyInstructionArgument() any
}

// InstructionReference describes one symbol-bearing value in a typed instruction argument.
type InstructionReference struct {
	Value         Node
	Modifiers     []Modifier
	ReferenceType ReferenceType
}

// InstructionReferenceProvider exposes symbol-bearing values from an opaque instruction argument.
type InstructionReferenceProvider interface {
	InstructionReferences() []InstructionReference
}

// InstructionFormProvider returns a stable target-specific operand and state form key.
type InstructionFormProvider interface {
	InstructionFormKey() string
}

// InstructionStateTransitionProvider returns a stable target-state transition form.
type InstructionStateTransitionProvider interface {
	InstructionStateTransitionForm() (string, bool)
}

// InstructionArgument stores an architecture-specific typed instruction argument value.
type InstructionArgument struct {
	*node

	Value any
}

// InstructionArguments stores multiple instruction operands in source order.
type InstructionArguments struct {
	*node

	Values []Node
}

// NewInstructionArgument returns a new typed instruction argument. It panics
// when a mutable value does not implement InstructionArgumentCopier.
func NewInstructionArgument(value any) InstructionArgument {
	validateInstructionArgumentValue(value)
	return InstructionArgument{
		node:  &node{},
		Value: value,
	}
}

// NewInstructionArguments returns a new instruction argument list node.
func NewInstructionArguments(values ...Node) InstructionArguments {
	return InstructionArguments{
		node:   &node{},
		Values: values,
	}
}

// InstructionArgumentForm returns a stable form key without literal values or symbol names.
func InstructionArgumentForm(argument Node) string {
	switch value := argument.(type) {
	case nil:
		return "none"
	case InstructionArgument:
		return instructionArgumentValueForm(value.Value)
	case *InstructionArgument:
		if value == nil {
			return "none"
		}
		return instructionArgumentValueForm(value.Value)
	case InstructionArguments:
		forms := make([]string, len(value.Values))
		for index, item := range value.Values {
			forms[index] = InstructionArgumentForm(item)
		}
		return "list[" + strings.Join(forms, ",") + "]"
	case *InstructionArguments:
		if value == nil {
			return "none"
		}
		return InstructionArgumentForm(*value)
	default:
		return fmt.Sprintf("%T", argument)
	}
}

// InstructionStateTransitionForm returns the target-state transition for one typed argument.
func InstructionStateTransitionForm(argument Node) (string, bool) {
	switch value := argument.(type) {
	case InstructionArgument:
		return instructionArgumentValueStateTransitionForm(value.Value)
	case *InstructionArgument:
		if value == nil {
			return "", false
		}
		return instructionArgumentValueStateTransitionForm(value.Value)
	default:
		return "", false
	}
}

func instructionArgumentValueForm(value any) string {
	if instructionArgumentValueIsNil(value) {
		return "none"
	}
	if provider, ok := value.(InstructionFormProvider); ok {
		return provider.InstructionFormKey()
	}
	return fmt.Sprintf("%T", value)
}

func instructionArgumentValueStateTransitionForm(value any) (string, bool) {
	if instructionArgumentValueIsNil(value) {
		return "", false
	}
	provider, ok := value.(InstructionStateTransitionProvider)
	if !ok {
		return "", false
	}
	return provider.InstructionStateTransitionForm()
}

// Copy returns a copy of the instruction argument node.
func (a InstructionArgument) Copy() Node {
	value := copyInstructionArgumentValue(a.Value)
	return InstructionArgument{
		node:  a.node.copyNode(),
		Value: value,
	}
}

func copyInstructionArgumentValue(value any) any {
	validateInstructionArgumentValue(value)
	if instructionArgumentValueIsNil(value) {
		return value
	}
	if copier, ok := value.(InstructionArgumentCopier); ok {
		return copier.CopyInstructionArgument()
	}
	return value
}

func validateInstructionArgumentValue(value any) {
	if instructionArgumentValueIsNil(value) {
		return
	}
	if _, ok := value.(InstructionArgumentCopier); ok {
		return
	}
	if !instructionArgumentValueIsImmutable(reflect.ValueOf(value)) {
		panic(fmt.Sprintf("ast: mutable instruction argument %T must implement InstructionArgumentCopier", value))
	}
}

func instructionArgumentValueIsNil(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

func instructionArgumentValueIsImmutable(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}

	switch value.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
		reflect.String:
		return true
	case reflect.Array:
		for index := range value.Len() {
			if !instructionArgumentValueIsImmutable(value.Index(index)) {
				return false
			}
		}
		return true
	case reflect.Struct:
		for index := range value.NumField() {
			if !instructionArgumentValueIsImmutable(value.Field(index)) {
				return false
			}
		}
		return true
	case reflect.Interface:
		return value.IsNil() || instructionArgumentValueIsImmutable(value.Elem())
	default:
		return false
	}
}

// CopyNodes deeply copies an AST node slice.
func CopyNodes(nodes []Node) []Node {
	if nodes == nil {
		return nil
	}
	copied := make([]Node, len(nodes))
	for index, node := range nodes {
		if node != nil {
			copied[index] = node.Copy()
		}
	}
	return copied
}

// Copy returns a copy of the instruction argument list node.
func (a InstructionArguments) Copy() Node {
	return InstructionArguments{
		node:   a.node.copyNode(),
		Values: CopyNodes(a.Values),
	}
}
