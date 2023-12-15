package ast

// Number ...
type Number struct {
	Value uint64
}

func (n Number) node() {}
