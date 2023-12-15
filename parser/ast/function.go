package ast

// Function ...
type Function struct {
	Name string

	Comment Comment
}

func (f *Function) node() {}

// SetComment sets the comment for the node.
func (f *Function) SetComment(message string) {
	f.Comment.Message = message
}

// FunctionEnd ...
type FunctionEnd struct {
}

func (f FunctionEnd) node() {}
