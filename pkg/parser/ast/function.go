package ast

// Function represents a procedure/function definition (.proc).
type Function struct {
	*node

	Name string
}

// FunctionEnd represents the end of a procedure/function (.endproc).
type FunctionEnd struct {
	*node
}

// NewFunction returns a new function node.
func NewFunction(name string) Function {
	return Function{
		node: &node{},
		Name: name,
	}
}

// NewFunctionEnd returns a new function end node.
func NewFunctionEnd() FunctionEnd {
	return FunctionEnd{
		node: &node{},
	}
}

// Copy returns a copy of the function node.
func (f Function) Copy() Node {
	return Function{
		node: f.node,
		Name: f.Name,
	}
}

// Copy returns a copy of the function end node.
func (f FunctionEnd) Copy() Node {
	return FunctionEnd{
		node: f.node,
	}
}
