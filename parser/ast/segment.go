package ast

// Segment ...
type Segment struct {
	*node

	Name string
}

// NewSegment returns a new segment node.
func NewSegment(name string) Segment {
	return Segment{
		node: &node{},
		Name: name,
	}
}
