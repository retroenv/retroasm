package ast

// RegisterValue represents a single register paired with a value node.
type RegisterValue struct {
	*node

	Register byte
	Value    Node
}

// RegisterRegisterValue represents two registers paired with a value node.
type RegisterRegisterValue struct {
	*node

	Register1 byte
	Register2 byte
	Value     Node
}

// NewRegisterValue returns a new register-value node.
func NewRegisterValue(register byte, value Node) RegisterValue {
	return RegisterValue{
		node:     &node{},
		Register: register,
		Value:    value,
	}
}

// NewRegisterRegisterValue returns a new register-register-value node.
func NewRegisterRegisterValue(register1, register2 byte, value Node) RegisterRegisterValue {
	return RegisterRegisterValue{
		node:      &node{},
		Register1: register1,
		Register2: register2,
		Value:     value,
	}
}

// Copy returns a copy of the register-value node.
func (r RegisterValue) Copy() Node {
	var value Node
	if r.Value != nil {
		value = r.Value.Copy()
	}

	return RegisterValue{
		node:     r.node,
		Register: r.Register,
		Value:    value,
	}
}

// Copy returns a copy of the register-register-value node.
func (r RegisterRegisterValue) Copy() Node {
	var value Node
	if r.Value != nil {
		value = r.Value.Copy()
	}

	return RegisterRegisterValue{
		node:      r.node,
		Register1: r.Register1,
		Register2: r.Register2,
		Value:     value,
	}
}
