package ast

// Variable ...
type Variable struct {
	Name             string
	Size             int
	UseOffsetCounter bool // TODO support

	Comment Comment
}

func (v *Variable) node() {}

// SetComment sets the comment for the node.
func (v *Variable) SetComment(message string) {
	v.Comment.Message = message
}
