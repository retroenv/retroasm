package ast

// Comment ...
type Comment struct {
	Message string
}

func (c Comment) node() {}
