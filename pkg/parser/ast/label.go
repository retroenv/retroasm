package ast

// Label represents a named location in the assembly program.
type Label struct {
	*node

	Name string
}

// NewLabel returns a new label node.
func NewLabel(name string) Label {
	return Label{
		node: &node{},
		Name: name,
	}
}

// Copy returns a copy of the label node.
func (l Label) Copy() Node {
	return Label{
		node: l.node,
		Name: l.Name,
	}
}
