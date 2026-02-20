package ast

// Segment represents a code or data segment directive (.segment).
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

// Copy returns a copy of the segment node.
func (s Segment) Copy() Node {
	return Segment{
		node: s.node,
		Name: s.Name,
	}
}
