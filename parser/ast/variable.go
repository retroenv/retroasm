package ast

// Variable ...
type Variable struct {
	node

	Name             string
	Size             int
	UseOffsetCounter bool // TODO support
}
