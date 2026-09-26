package ast

import "slices"

// Equal compares all node fields with the same rules as reflect.DeepEqual.
// It preserves dynamic types, source comments, and nil/empty distinctions.
// Operational entry handles have no assembly meaning and are ignored.
// Complex operands and extension nodes use reflection to support cycles.
func Equal(left, right Node) bool {
	switch before := left.(type) {
	case Instruction:
		after, ok := right.(Instruction)
		return ok && equalInstruction(before, after)
	case *Instruction:
		after, ok := right.(*Instruction)
		return ok && equalPointer(before, after, equalInstruction)
	default:
		return equalLeaf(left, right)
	}
}

func equalInstruction(left, right Instruction) bool {
	return equalBase(left.node, right.node) && left.OpcodeID == right.OpcodeID &&
		left.Name == right.Name && left.Addressing == right.Addressing &&
		equalLeaf(left.Argument, right.Argument) && equalModifiers(left.Modifier, right.Modifier)
}

func equalLeaf(left, right Node) bool {
	// Do not recurse through composite nodes here. Reflection owns cycle detection.
	switch before := left.(type) {
	case nil:
		return right == nil
	case Number:
		after, ok := right.(Number)
		return ok && equalNumber(before, after)
	case Label:
		after, ok := right.(Label)
		return ok && equalLabel(before, after)
	case Identifier:
		after, ok := right.(Identifier)
		return ok && equalIdentifier(before, after)
	case Operator:
		after, ok := right.(Operator)
		return ok && equalOperator(before, after)
	default:
		return equalLeafPointer(left, right)
	}
}

func equalLeafPointer(left, right Node) bool {
	switch before := left.(type) {
	case *Number:
		after, ok := right.(*Number)
		return ok && equalPointer(before, after, equalNumber)
	case *Label:
		after, ok := right.(*Label)
		return ok && equalPointer(before, after, equalLabel)
	case *Identifier:
		after, ok := right.(*Identifier)
		return ok && equalPointer(before, after, equalIdentifier)
	case *Operator:
		after, ok := right.(*Operator)
		return ok && equalPointer(before, after, equalOperator)
	case *Comment:
		after, ok := right.(*Comment)
		return ok && (before == after || before != nil && after != nil && before.Message == after.Message)
	default:
		return equalComposite(left, right)
	}
}

func equalPointer[T any](left, right *T, equal func(T, T) bool) bool {
	return left == right || left != nil && right != nil && equal(*left, *right)
}

func equalBase(left, right *node) bool {
	return left == right || left != nil && right != nil && left.comment.Message == right.comment.Message
}

func equalNumber(left, right Number) bool {
	return equalBase(left.node, right.node) && left.Value == right.Value
}

func equalLabel(left, right Label) bool {
	return equalBase(left.node, right.node) && left.Name == right.Name
}

func equalIdentifier(left, right Identifier) bool {
	return equalBase(left.node, right.node) && left.Name == right.Name &&
		(left.Arguments == nil) == (right.Arguments == nil) && slices.Equal(left.Arguments, right.Arguments)
}

func equalOperator(left, right Operator) bool {
	return equalBase(left.node, right.node) && left.Operator == right.Operator
}

func equalModifiers(left, right []Modifier) bool {
	if (left == nil) != (right == nil) || len(left) != len(right) {
		return false
	}

	for index, before := range left {
		after := right[index]
		if !equalBase(&before.node, &after.node) || before.Value != after.Value || !equalOperator(before.Operator, after.Operator) {
			return false
		}
	}
	return true
}
