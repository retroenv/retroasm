package ast

// Comment represents an inline or standalone comment in the assembly source.
type Comment struct {
	// Message is the source comment text.
	Message string

	handle *entryHandle
}

// SetComment sets the comment for the node.
func (c *Comment) SetComment(message string) {
	c.Message = message
}

// Copy returns a copy of the comment node.
func (c *Comment) Copy() Node {
	return &Comment{
		Message: c.Message,
		handle:  c.handle,
	}
}

func (c *Comment) entryHandle() *entryHandle          { return c.handle }
func (c *Comment) setEntryHandle(handle *entryHandle) { c.handle = handle }
