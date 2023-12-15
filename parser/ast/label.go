package ast

// Label ...
type Label struct {
	Name string

	Comment Comment
}

func (l *Label) node() {}

// SetComment sets the comment for the node.
func (l *Label) SetComment(message string) {
	l.Comment.Message = message
}
