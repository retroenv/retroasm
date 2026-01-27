package ast

// Comment ...
type Comment struct {
	Message string
}

// SetComment sets the comment for the node.
func (c *Comment) SetComment(message string) {
	c.Message = message
}

// Copy returns a copy of the comment node.
func (c *Comment) Copy() Node {
	return &Comment{
		Message: c.Message,
	}
}
