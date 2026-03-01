package ast

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
	return InstructionArgument{
		node:  a.node,
		Value: a.Value,
	}
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
