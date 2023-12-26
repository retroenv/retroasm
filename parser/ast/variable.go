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
