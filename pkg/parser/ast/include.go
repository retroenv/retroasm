package ast

// Include ...
type Include struct {
	*node

	Name   string
	Binary bool

	Start int
	Size  int
}

// NewInclude returns a new include node.
func NewInclude(name string, binary bool, start, size int) Include {
	return Include{
		node: &node{},

		Name:   name,
		Binary: binary,
		Start:  start,
		Size:   size,
	}
}

// Copy returns a copy of the include node.
func (i Include) Copy() Node {
	return Include{
		node:   i.node,
		Name:   i.Name,
		Binary: i.Binary,
		Start:  i.Start,
		Size:   i.Size,
	}
}
