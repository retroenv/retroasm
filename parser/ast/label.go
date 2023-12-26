package ast

// Label ...
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
