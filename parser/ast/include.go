package ast

// Include ...
type Include struct {
	Name   string
	Binary bool

	Start int
	Size  int

	Comment Comment
}

func (i *Include) node() {}

// SetComment sets the comment for the node.
func (i *Include) SetComment(message string) {
	i.Comment.Message = message
}
