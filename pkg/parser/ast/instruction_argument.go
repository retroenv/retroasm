package ast

// InstructionArgumentCopier defines deep-copy behavior for an opaque typed
// instruction argument. Mutable architecture-specific values must implement it.
type InstructionArgumentCopier interface {
	CopyInstructionArgument() any
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

// NewInstructionArgument returns a new typed instruction argument.
func NewInstructionArgument(value any) InstructionArgument {
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

// Copy returns a copy of the instruction argument node.
func (a InstructionArgument) Copy() Node {
	value := a.Value
	if copier, ok := value.(InstructionArgumentCopier); ok {
		value = copier.CopyInstructionArgument()
	}
	return InstructionArgument{
		node:  a.node,
		Value: value,
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
	values := make([]Node, 0, len(a.Values))
	for _, value := range a.Values {
		values = append(values, value.Copy())
	}

	return InstructionArguments{
		node:   a.node,
		Values: values,
	}
}
