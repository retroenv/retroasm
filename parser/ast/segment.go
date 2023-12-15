package ast

// Segment ...
type Segment struct {
	Name string

	Comment Comment
}

func (seg *Segment) node() {}

// SetComment sets the comment for the node.
func (seg *Segment) SetComment(message string) {
	seg.Comment.Message = message
}
