package ast

// Bank ...
type Bank struct {
	Number int

	Comment Comment
}

func (b *Bank) node() {}

// SetComment sets the comment for the node.
func (b *Bank) SetComment(message string) {
	b.Comment.Message = message
}
