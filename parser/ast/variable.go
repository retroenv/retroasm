package ast

// Variable ...
type Variable struct {
	*node

	Name             string
	Size             int
	UseOffsetCounter bool // TODO support
}

// NewVariable returns a new variable node.
func NewVariable(name string, size int) Variable {
	return Variable{
		node: &node{},
		Name: name,
		Size: size,
	}
}

// Copy returns a copy of the variable node.
func (v Variable) Copy() Node {
	return Variable{
		node:             v.node,
		Name:             v.Name,
		Size:             v.Size,
		UseOffsetCounter: v.UseOffsetCounter,
	}
}
