package ast

// Comment ...
type Comment struct {
	Message string
}

func (c Comment) astNode() {}

// SetComment sets the comment for the node.
func (c *Comment) SetComment(message string) {
	c.Message = message
}
