package ast

// Function ...
type Function struct {
	*node

	Name string
}

// NewFunction returns a new function node.
func NewFunction(name string) Function {
	return Function{
		node: &node{},
		Name: name,
	}
}

// FunctionEnd ...
type FunctionEnd struct {
	*node
}

// NewFunctionEnd returns a new function end node.
func NewFunctionEnd() FunctionEnd {
	return FunctionEnd{
		node: &node{},
	}
}
