package ast

// Scope represents a named scope definition (.scope).
// Unlike Function (.proc), Scope does not create an entry-point label.
type Scope struct {
	*node

	Name string
}

// ScopeEnd represents the end of a scope (.endscope).
type ScopeEnd struct {
	*node
}

// NewScope returns a new scope node.
func NewScope(name string) Scope {
	return Scope{
		node: &node{},
		Name: name,
	}
}

// NewScopeEnd returns a new scope end node.
func NewScopeEnd() ScopeEnd {
	return ScopeEnd{
		node: &node{},
	}
}

// Copy returns a copy of the scope node.
func (s Scope) Copy() Node {
	return Scope{
		node: s.node,
		Name: s.Name,
	}
}

// Copy returns a copy of the scope end node.
func (s ScopeEnd) Copy() Node {
	return ScopeEnd{
		node: s.node,
	}
}
