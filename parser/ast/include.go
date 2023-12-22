package ast

// Include ...
type Include struct {
	node

	Name   string
	Binary bool

	Start int
	Size  int
}
