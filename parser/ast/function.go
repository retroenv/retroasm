package ast

// Function ...
type Function struct {
	node

	Name string
}

// FunctionEnd ...
type FunctionEnd struct {
	*node
}
