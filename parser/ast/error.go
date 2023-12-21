package ast

// Error ...
type Error struct {
	Message string

	Comment Comment
}

func (e *Error) node() {}

// SetComment sets the comment for the node.
func (e *Error) SetComment(message string) {
	e.Comment.Message = message
}
